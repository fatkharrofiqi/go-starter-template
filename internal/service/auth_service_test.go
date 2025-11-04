package service

import (
	"context"
	"crypto"
	"errors"
	"io"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"go-starter-template/internal/config/env"
	"go-starter-template/internal/constant"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/errcode"
)

// helper: default env config for JWT durations/secrets
func testEnvConfig() *env.Config {
	cfg := &env.Config{}
	cfg.JWT.Secret = "access-secret"
	cfg.JWT.RefreshSecret = "refresh-secret"
	cfg.JWT.AccessTokenExpiration = 60
	cfg.JWT.RefreshTokenExpiration = 120
	return cfg
}

// helper: silent logger
func testLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// helper: setup sqlmock-backed UserRepository and UnitOfWork
func setupRepoAndUow(t *testing.T) (*repository.UserRepository, *repository.UnitOfWork, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	repo := repository.NewUserRepository(db)
	uow := repository.NewUnitOfWork(db)
	cleanup := func() { _ = db.Close() }
	return repo, uow, mock, cleanup
}

// fake blacklist repository implementing interface
type fakeBLRepo struct {
	isBlacklisted func(tokenHash string, tokenType constant.TokenType) (bool, error)
	add           func(tokenHash string, tokenType constant.TokenType, d time.Duration) error
}

// test-scoped global to store original HS256 hash when overriding in a case
var jwtSigningHashOrig crypto.Hash

func (f *fakeBLRepo) Add(token string, tokenType constant.TokenType, d time.Duration) error {
	if f.add != nil {
		return f.add(token, tokenType, d)
	}
	return nil
}
func (f *fakeBLRepo) IsBlacklisted(token string, tokenType constant.TokenType) (bool, error) {
	if f.isBlacklisted != nil {
		return f.isBlacklisted(token, tokenType)
	}
	return false, nil
}

// signing method that always fails when signing (used to force refresh token generation errors)
type failingSignMethod struct{}

func (failingSignMethod) Alg() string { return "HS256" }
func (failingSignMethod) Sign(signingString string, key interface{}) ([]byte, error) {
	return nil, errors.New("forced sign error")
}
func (failingSignMethod) Verify(signingString string, signature []byte, key interface{}) error {
	return errors.New("forced verify error")
}

// Login tests
func TestAuthService_Login(t *testing.T) {
	type testcase struct {
		name    string
		setupDB func(sqlmock.Sqlmock)
		req     *dto.LoginRequest
		assert  func(*testing.T, string, string, error)
		before  func(*JwtService)
		after   func(*JwtService)
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)

	cases := []testcase{
		{
			name: "Success",
			req:  &dto.LoginRequest{Email: "user@example.com", Password: "pass"},
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`)).
					WithArgs("user@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
						AddRow("u1", "Name", "user@example.com", string(hashed), time.Now(), time.Now()))
			},
			assert: func(t *testing.T, access, refresh string, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, access)
				require.NotEmpty(t, refresh)
			},
		},
		{
			name: "UserNotFound",
			req:  &dto.LoginRequest{Email: "missing@example.com", Password: "pass"},
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`)).
					WithArgs("missing@example.com").
					WillReturnError(errors.New("no rows"))
			},
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorIs(t, err, errcode.ErrInvalidEmailOrPassword)
			},
		},
		{
			name: "InvalidPassword",
			req:  &dto.LoginRequest{Email: "user@example.com", Password: "wrong"},
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`)).
					WithArgs("user@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
						AddRow("u1", "Name", "user@example.com", string(hashed), time.Now(), time.Now()))
			},
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorIs(t, err, errcode.ErrInvalidEmailOrPassword)
			},
		},
		{
			name: "AccessTokenSignMethodError",
			req:  &dto.LoginRequest{Email: "user@example.com", Password: "pass"},
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`)).
					WithArgs("user@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
						AddRow("u1", "Name", "user@example.com", string(hashed), time.Now(), time.Now()))
			},
			before: func(_ *JwtService) {
				// Force HS256 to use an unavailable hash to make SignedString fail
				origHash := jwt.SigningMethodHS256.Hash
				jwt.SigningMethodHS256.Hash = crypto.Hash(0)
				// stash original in closure variable
				jwtSigningHashOrig = origHash
			},
			after: func(_ *JwtService) {
				// Restore original hash method
				if jwtSigningHashOrig != 0 {
					jwt.SigningMethodHS256.Hash = jwtSigningHashOrig
				}
			},
			assert: func(t *testing.T, access, refresh string, err error) {
				require.ErrorIs(t, err, errcode.ErrAccessTokenGeneration)
				require.Empty(t, access)
				require.Empty(t, refresh)
			},
		},
		{
			name: "RefreshTokenSignMethodError",
			req:  &dto.LoginRequest{Email: "user@example.com", Password: "pass"},
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`)).
					WithArgs("user@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
						AddRow("u1", "Name", "user@example.com", string(hashed), time.Now(), time.Now()))
			},
			before: func(js *JwtService) {
				// Override only the refresh signing method to force a signing error
				js.SetRefreshMethod(failingSignMethod{})
			},
			after: func(js *JwtService) {
				// Restore to default HS256
				js.SetRefreshMethod(jwt.SigningMethodHS256)
			},
			assert: func(t *testing.T, access, refresh string, err error) {
				require.ErrorIs(t, err, errcode.ErrRefreshTokenGeneration)
				// On error, Login returns empty tokens
				require.Empty(t, access)
				require.Empty(t, refresh)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, uow, mock, cleanup := setupRepoAndUow(t)
			defer cleanup()
			cfg := testEnvConfig()
			log := testLogger()
			jwtSvc := NewJwtService(log, cfg)
			blSvc := NewBlacklistService(log, jwtSvc, &fakeBLRepo{})
			svc := NewAuthService(jwtSvc, repo, blSvc, log, uow)

			if tc.setupDB != nil {
				tc.setupDB(mock)
			}
			if tc.before != nil {
				tc.before(jwtSvc)
			}

			access, refresh, err := svc.Login(context.Background(), tc.req)

			if tc.assert != nil {
				tc.assert(t, access, refresh, err)
			}
			if tc.after != nil {
				tc.after(jwtSvc)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Register tests (transactional)
func TestAuthService_Register(t *testing.T) {
	type testcase struct {
		name       string
		setupDB    func(sqlmock.Sqlmock)
		expectErr  error
		assertResp func(*testing.T, *dto.UserResponse)
		mutateSvc  func(*AuthService)
		invoke     func(*testing.T, *AuthService)
	}

	cases := []testcase{
		{
			name: "DBError_CheckingExisting",
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM users WHERE email = $1`)).
					WithArgs("new@example.com").
					WillReturnError(errors.New("db error"))
				mock.ExpectRollback()
			},
			expectErr: errcode.ErrDatabaseError,
		},
		{
			name: "AlreadyExists",
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM users WHERE email = $1`)).
					WithArgs("new@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectRollback()
			},
			expectErr: errcode.ErrUserAlreadyExists,
		},
		{
			name: "CreateError",
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM users WHERE email = $1`)).
					WithArgs("new@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO users (uuid, name, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
    `)).WillReturnError(errors.New("create error"))
				mock.ExpectRollback()
			},
			expectErr: errcode.ErrUserCreationFailed,
		},
		{
			name: "Success",
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM users WHERE email = $1`)).
					WithArgs("new@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO users (uuid, name, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
    `)).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			assertResp: func(t *testing.T, resp *dto.UserResponse) {
				require.NotNil(t, resp)
				require.Equal(t, "new@example.com", resp.Email)
				require.Equal(t, "New User", resp.Name)
				require.NotEmpty(t, resp.UUID)
			},
		},
		{
			name: "PasswordHashError",
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM users WHERE email = $1`)).
					WithArgs("new@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				// No INSERT expected because hashing fails
				mock.ExpectRollback()
			},
			expectErr: errcode.ErrPasswordEncryption,
			mutateSvc: func(s *AuthService) {
				s.hashPassword = func(_ []byte, _ int) ([]byte, error) { return nil, errors.New("hash fail") }
			},
		},
		{
			name: "PanicRecovery",
			setupDB: func(mock sqlmock.Sqlmock) {
				// BeginTx will be called by UnitOfWork.Do before panic occurs
				mock.ExpectBegin()
				// No further expectations; panic will abort and rollback is not reached because we recover at service level
			},
			mutateSvc: func(s *AuthService) {
				// Induce panic inside UOW callback by using nil repository
				s.userRepository = nil
			},
			invoke: func(t *testing.T, svc *AuthService) {
				require.NotPanics(t, func() {
					_, _ = svc.Register(context.Background(), &dto.RegisterRequest{Email: "panic@example.com", Password: "x", Name: "Panic"})
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, uow, mock, cleanup := setupRepoAndUow(t)
			defer cleanup()
			cfg := testEnvConfig()
			log := testLogger()
			jwtSvc := NewJwtService(log, cfg)
			blSvc := NewBlacklistService(log, jwtSvc, &fakeBLRepo{})
			svc := NewAuthService(jwtSvc, repo, blSvc, log, uow)
			if tc.mutateSvc != nil {
				tc.mutateSvc(svc)
			}

			if tc.setupDB != nil {
				tc.setupDB(mock)
			}
			if tc.invoke != nil {
				tc.invoke(t, svc)
			} else {
				resp, err := svc.Register(context.Background(), &dto.RegisterRequest{Email: "new@example.com", Password: "password123", Name: "New User"})

				if tc.expectErr != nil {
					require.ErrorIs(t, err, tc.expectErr)
				} else {
					require.NoError(t, err)
					tc.assertResp(t, resp)
				}
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Register panic recovery: ensure no panic when repository is nil
// (converted into table-driven case in TestAuthService_Register)

// RefreshToken tests
func TestAuthService_RefreshToken(t *testing.T) {
	type testcase struct {
		name      string
		setupRepo func(*fakeBLRepo)
		mutateSvc func(*JwtService)
		after     func(*JwtService)
		token     string
		assert    func(*testing.T, string, string, error)
	}

	cfg := testEnvConfig()
	log := testLogger()
	jwtSvc := NewJwtService(log, cfg)

	validRefresh, err := jwtSvc.GenerateRefreshToken(context.Background(), "u1")
	require.NoError(t, err)

	cases := []testcase{
		{
			name: "Blacklisted",
			setupRepo: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return true, nil }
			},
			token: validRefresh,
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorIs(t, err, errcode.ErrUnauthorized)
			},
		},
		{
			name:  "InvalidToken",
			token: "bad-token",
			setupRepo: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, nil }
			},
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorIs(t, err, errcode.ErrInvalidToken)
			},
		},
		{
			name:  "Success",
			token: validRefresh,
			setupRepo: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, nil }
				f.add = func(_ string, _ constant.TokenType, d time.Duration) error {
					require.True(t, d > 0)
					return nil
				}
			},
			assert: func(t *testing.T, access, newRefresh string, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, access)
				require.NotEmpty(t, newRefresh)
			},
		},
		{
			name:  "AccessGenerationError",
			token: validRefresh,
			setupRepo: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, nil }
			},
			mutateSvc: func(js *JwtService) {
				js.SetAccessMethod(failingSignMethod{})
			},
			after: func(js *JwtService) {
				js.SetAccessMethod(jwt.SigningMethodHS256)
			},
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorIs(t, err, errcode.ErrAccessTokenGeneration)
			},
		},
		{
			name:  "RefreshGenerationError",
			token: validRefresh,
			setupRepo: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, nil }
			},
			mutateSvc: func(js *JwtService) {
				js.SetRefreshMethod(failingSignMethod{})
			},
			after: func(js *JwtService) {
				js.SetRefreshMethod(jwt.SigningMethodHS256)
			},
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorIs(t, err, errcode.ErrRefreshTokenGeneration)
			},
		},
		{
			name:  "BlacklistAddFails",
			token: validRefresh,
			setupRepo: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, nil }
				f.add = func(_ string, _ constant.TokenType, _ time.Duration) error { return errors.New("redis set") }
			},
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorIs(t, err, errcode.ErrRedisSet)
			},
		},
		{
			name:  "IsTokenBlacklistedError",
			token: validRefresh,
			setupRepo: func(f *fakeBLRepo) {
				f.isBlacklisted = func(_ string, _ constant.TokenType) (bool, error) { return false, errors.New("redis get") }
			},
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorIs(t, err, errcode.ErrUnauthorized)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeBLRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(f)
			}
			blSvc := NewBlacklistService(log, jwtSvc, f)
			svc := NewAuthService(jwtSvc, nil, blSvc, log, nil)
			if tc.mutateSvc != nil {
				tc.mutateSvc(jwtSvc)
			}

			access, refresh, err := svc.RefreshToken(context.Background(), tc.token)
			tc.assert(t, access, refresh, err)
			if tc.after != nil {
				tc.after(jwtSvc)
			}
		})
	}
}

// (moved BlacklistAddFails and IsTokenBlacklistedError into the main RefreshToken table)

// Logout tests
func TestAuthService_Logout(t *testing.T) {
	type testcase struct {
		name      string
		setupRepo func(*fakeBLRepo)
		assert    func(*testing.T, error)
	}

	cases := []testcase{
		{
			name: "Success",
			setupRepo: func(f *fakeBLRepo) {
				f.add = func(_ string, _ constant.TokenType, _ time.Duration) error { return nil }
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "AddError",
			setupRepo: func(f *fakeBLRepo) {
				f.add = func(_ string, tt constant.TokenType, _ time.Duration) error {
					if tt == constant.TokenTypeAccess {
						return errors.New("add failed")
					}
					return nil
				}
			},
			assert: func(t *testing.T, err error) { require.Error(t, err) },
		},
	}

	cfg := testEnvConfig()
	log := testLogger()
	jwtSvc := NewJwtService(log, cfg)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeBLRepo{}
			if tc.setupRepo != nil {
				tc.setupRepo(f)
			}
			blSvc := NewBlacklistService(log, jwtSvc, f)
			svc := NewAuthService(jwtSvc, nil, blSvc, log, nil)

			err := svc.Logout(context.Background(), "access", "refresh")
			tc.assert(t, err)
		})
	}
}
