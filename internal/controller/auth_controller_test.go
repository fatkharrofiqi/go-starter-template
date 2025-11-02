package controller

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"go-starter-template/internal/config/env"
	"go-starter-template/internal/config/validation"
	"go-starter-template/internal/constant"
	"go-starter-template/internal/dto"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/service"
	"go-starter-template/internal/utils/errcode"
)

// noopBlacklistRepo is a trivial implementation of TokenBlacklistRepository for wiring services.
type noopBlacklistRepo struct{}

func (n *noopBlacklistRepo) Add(token string, tokenType constant.TokenType, duration time.Duration) error {
	return nil
}

func (n *noopBlacklistRepo) IsBlacklisted(token string, tokenType constant.TokenType) (bool, error) {
	return false, nil
}

// setupControllerWithMock prepares an AuthController with sqlmock for tests.
func setupControllerWithMock(t *testing.T) (*AuthController, *fiber.App, sqlmock.Sqlmock) {
	t.Helper()

	// SQL mock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	cfg := &env.Config{}
	cfg.JWT.Secret = "access_secret"
	cfg.JWT.RefreshSecret = "refresh_secret"
	cfg.JWT.AccessTokenExpiration = 60
	cfg.JWT.RefreshTokenExpiration = 120
	cfg.JWT.CsrfTokenExpiration = 900

	jwtService := service.NewJwtService(logger, cfg)
	blRepo := &noopBlacklistRepo{}
	blacklistService := service.NewBlacklistService(logger, jwtService, blRepo)
	userRepo := repository.NewUserRepository(db)
	uow := repository.NewUnitOfWork(db)
	authService := service.NewAuthService(jwtService, userRepo, blacklistService, logger, uow)

	validator := validation.NewValidation()
	ctrl := NewAuthController(authService, logger, validator, cfg)

	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error {
		// Treat validation errors as 400
		if _, ok := err.(*validation.ValidationError); ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if code, ok := errcode.GetHTTPStatus(err); ok {
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}})

	return ctrl, app, mock
}

// Table-driven test for Login
func TestAuthController_Login(t *testing.T) {
	type testcase struct {
		name         string
		setupMock    func(sqlmock.Sqlmock)
		body         string
		expectStatus int
		assert       func(*testing.T, *http.Response)
	}

	cases := []testcase{
		{
			name:         "InvalidJSON",
			body:         "{invalid}",
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "ValidationError",
			body:         `{"email":"not-an-email","password":"short"}`,
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Success",
			setupMock: func(mock sqlmock.Sqlmock) {
				hashed, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
				now := time.Now()
				query := `SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("john@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
						AddRow("user-123", "John Doe", "john@example.com", string(hashed), now, now))
			},
			body:         `{"email":"john@example.com","password":"secret123"}`,
			expectStatus: http.StatusOK,
			assert: func(t *testing.T, resp *http.Response) {
				var out dto.WebResponse[*dto.TokenResponse]
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				require.NotNil(t, out.Data)
				require.NotEmpty(t, out.Data.AccessToken)
				var refreshCookie *http.Cookie
				for _, c := range resp.Cookies() {
					if c.Name == refreshTokenCookieName {
						refreshCookie = c
						break
					}
				}
				require.NotNil(t, refreshCookie)
				require.NotEmpty(t, refreshCookie.Value)
				require.True(t, refreshCookie.HttpOnly)
				require.True(t, refreshCookie.Secure)
				require.Equal(t, "/", refreshCookie.Path)
			},
		},
		{
			name: "InvalidEmail",
			setupMock: func(mock sqlmock.Sqlmock) {
				query := `SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("missing@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			body:         `{"email":"missing@example.com","password":"whatever"}`,
			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "InvalidPassword",
			setupMock: func(mock sqlmock.Sqlmock) {
				hashed, _ := bcrypt.GenerateFromPassword([]byte("otherpass"), bcrypt.DefaultCost)
				now := time.Now()
				query := `SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("john@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
						AddRow("user-123", "John Doe", "john@example.com", string(hashed), now, now))
			},
			body:         `{"email":"john@example.com","password":"secret123"}`,
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl, app, mock := setupControllerWithMock(t)
			app.Post("/login", ctrl.Login)
			if tc.setupMock != nil {
				tc.setupMock(mock)
			}
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			require.Equal(t, tc.expectStatus, resp.StatusCode)
			if tc.assert != nil {
				tc.assert(t, resp)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Table-driven test for RefreshToken
func TestAuthController_RefreshToken(t *testing.T) {
	type testcase struct {
		name         string
		buildRequest func(*testing.T, *AuthController) *http.Request
		expectStatus int
		assert       func(*testing.T, *http.Response)
	}

	cases := []testcase{
		{
			name: "MissingCookie",
			buildRequest: func(t *testing.T, ctrl *AuthController) *http.Request {
				return httptest.NewRequest(http.MethodPost, "/refresh", nil)
			},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "InvalidCookie_ClearsCookie",
			buildRequest: func(t *testing.T, ctrl *AuthController) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
				req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "invalid"})
				return req
			},
			expectStatus: http.StatusUnauthorized,
			assert: func(t *testing.T, resp *http.Response) {
				var cleared *http.Cookie
				for _, c := range resp.Cookies() {
					if c.Name == refreshTokenCookieName {
						cleared = c
						break
					}
				}
				require.NotNil(t, cleared)
				require.Equal(t, "", cleared.Value)
			},
		},
		{
			name: "Success",
			buildRequest: func(t *testing.T, ctrl *AuthController) *http.Request {
				logger := logrus.New()
				logger.SetOutput(io.Discard)
				jwtSvc := service.NewJwtService(logger, ctrl.config)
				token, err := jwtSvc.GenerateRefreshToken(context.Background(), "user-123")
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
				req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: token})
				return req
			},
			expectStatus: http.StatusOK,
			assert: func(t *testing.T, resp *http.Response) {
				var setCookie *http.Cookie
				for _, c := range resp.Cookies() {
					if c.Name == refreshTokenCookieName && c.Value != "" {
						setCookie = c
						break
					}
				}
				require.NotNil(t, setCookie)
				require.NotEmpty(t, setCookie.Value)
				require.True(t, setCookie.HttpOnly)
				require.True(t, setCookie.Secure)
				require.Equal(t, "/", setCookie.Path)

				var out dto.WebResponse[*dto.TokenResponse]
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				require.NotNil(t, out.Data)
				require.NotEmpty(t, out.Data.AccessToken)
			},
		},
	}

	ctrl, app, _ := setupControllerWithMock(t)
	app.Post("/refresh", ctrl.RefreshToken)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.buildRequest(t, ctrl)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			require.Equal(t, tc.expectStatus, resp.StatusCode)

			if tc.assert != nil {
				tc.assert(t, resp)
			}
		})
	}
}

// Table-driven test for Register
func TestAuthController_Register(t *testing.T) {
	type testcase struct {
		name         string
		setupMock    func(sqlmock.Sqlmock)
		body         string
		expectStatus int
		assert       func(*testing.T, *http.Response)
	}

	countQuery := `SELECT COUNT(*) FROM users WHERE email = $1`
	insertQuery := `
        INSERT INTO users (uuid, name, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
    `

	cases := []testcase{
		{
			name:         "InvalidJSON",
			body:         "{invalid}",
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Success",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(countQuery)).
					WithArgs("new@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectExec(regexp.QuoteMeta(insertQuery)).
					WithArgs(sqlmock.AnyArg(), "New User", "new@example.com", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			body:         `{"name":"New User","email":"new@example.com","password":"newpass123"}`,
			expectStatus: http.StatusOK,
			assert: func(t *testing.T, resp *http.Response) {
				var out dto.WebResponse[*dto.UserResponse]
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				require.NotNil(t, out.Data)
				require.Equal(t, "New User", out.Data.Name)
				require.Equal(t, "new@example.com", out.Data.Email)
			},
		},
		{
			name: "EmailExists",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(countQuery)).
					WithArgs("existing@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectRollback()
			},
			body:         `{"name":"Duplicate User","email":"existing@example.com","password":"newpass123"}`,
			expectStatus: http.StatusConflict,
			assert: func(t *testing.T, resp *http.Response) {
				var out struct {
					Error string `json:"error"`
				}
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				require.Equal(t, "user already exists", out.Error)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl, app, mock := setupControllerWithMock(t)
			app.Post("/register", ctrl.Register)
			if tc.setupMock != nil {
				tc.setupMock(mock)
			}
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			require.Equal(t, tc.expectStatus, resp.StatusCode)
			if tc.assert != nil {
				tc.assert(t, resp)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Table-driven test covering all Logout scenarios
func TestAuthController_Logout(t *testing.T) {
	// Helper to setup default controller/app
	defaultSetup := func(t *testing.T) (*AuthController, *fiber.App) {
		ctrl, app, _ := setupControllerWithMock(t)
		app.Post("/logout", ctrl.Logout)
		return ctrl, app
	}

	// Helper to setup controller with failing blacklist repo
	failingSetup := func(t *testing.T) (*AuthController, *fiber.App) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)

		logger := logrus.New()
		logger.SetOutput(io.Discard)

		cfg := &env.Config{}
		cfg.JWT.Secret = "access_secret"
		cfg.JWT.RefreshSecret = "refresh_secret"
		cfg.JWT.AccessTokenExpiration = 60
		cfg.JWT.RefreshTokenExpiration = 120
		cfg.JWT.CsrfTokenExpiration = 900

		jwtService := service.NewJwtService(logger, cfg)
		blRepo := &failingBlacklistRepo{}
		blacklistService := service.NewBlacklistService(logger, jwtService, blRepo)
		userRepo := repository.NewUserRepository(db)
		uow := repository.NewUnitOfWork(db)
		authService := service.NewAuthService(jwtService, userRepo, blacklistService, logger, uow)

		validator := validation.NewValidation()
		ctrl := NewAuthController(authService, logger, validator, cfg)

		app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error {
			if _, ok := err.(*validation.ValidationError); ok {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			}
			if code, ok := errcode.GetHTTPStatus(err); ok {
				return c.Status(code).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}})

		app.Post("/logout", ctrl.Logout)
		return ctrl, app
	}

	type testcase struct {
		name         string
		setup        func(*testing.T) (*AuthController, *fiber.App)
		buildRequest func(*testing.T, *AuthController) *http.Request
		expectStatus int
		assert       func(*testing.T, *http.Response)
	}

	cases := []testcase{
		{
			name:  "MissingAuthorization",
			setup: defaultSetup,
			buildRequest: func(t *testing.T, ctrl *AuthController) *http.Request {
				return httptest.NewRequest(http.MethodPost, "/logout", nil)
			},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:  "NonBearerAuthorization",
			setup: defaultSetup,
			buildRequest: func(t *testing.T, ctrl *AuthController) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/logout", nil)
				req.Header.Set("Authorization", "Basic abc123")
				return req
			},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:  "EmptyAccessToken",
			setup: defaultSetup,
			buildRequest: func(t *testing.T, ctrl *AuthController) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/logout", nil)
				req.Header.Set("Authorization", bearerPrefix)
				return req
			},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:  "MissingRefreshCookie",
			setup: defaultSetup,
			buildRequest: func(t *testing.T, ctrl *AuthController) *http.Request {
				logger := logrus.New()
				logger.SetOutput(io.Discard)
				jwtSvc := service.NewJwtService(logger, ctrl.config)
				accessToken, err := jwtSvc.GenerateAccessToken(context.Background(), "user-123")
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/logout", nil)
				req.Header.Set("Authorization", bearerPrefix+accessToken)
				return req
			},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:  "BlacklistError",
			setup: failingSetup,
			buildRequest: func(t *testing.T, ctrl *AuthController) *http.Request {
				logger := logrus.New()
				logger.SetOutput(io.Discard)
				jwtSvc := service.NewJwtService(logger, ctrl.config)
				accessToken, err := jwtSvc.GenerateAccessToken(context.Background(), "user-123")
				require.NoError(t, err)
				refreshToken, err := jwtSvc.GenerateRefreshToken(context.Background(), "user-123")
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/logout", nil)
				req.Header.Set("Authorization", bearerPrefix+accessToken)
				req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: refreshToken})
				return req
			},
			expectStatus: http.StatusInternalServerError,
			assert: func(t *testing.T, resp *http.Response) {
				// Ensure cookie not cleared (no empty cookie value)
				for _, c := range resp.Cookies() {
					require.NotEqual(t, refreshTokenCookieName, c.Name)
				}
			},
		},
		{
			name:  "Success",
			setup: defaultSetup,
			buildRequest: func(t *testing.T, ctrl *AuthController) *http.Request {
				logger := logrus.New()
				logger.SetOutput(io.Discard)
				jwtSvc := service.NewJwtService(logger, ctrl.config)
				accessToken, err := jwtSvc.GenerateAccessToken(context.Background(), "user-123")
				require.NoError(t, err)
				refreshToken, err := jwtSvc.GenerateRefreshToken(context.Background(), "user-123")
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/logout", nil)
				req.Header.Set("Authorization", bearerPrefix+accessToken)
				req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: refreshToken})
				return req
			},
			expectStatus: http.StatusOK,
			assert: func(t *testing.T, resp *http.Response) {
				var cleared *http.Cookie
				for _, c := range resp.Cookies() {
					if c.Name == refreshTokenCookieName {
						cleared = c
						break
					}
				}
				require.NotNil(t, cleared)
				require.Equal(t, "", cleared.Value)
				require.True(t, cleared.HttpOnly)
				require.True(t, cleared.Secure)
				require.Equal(t, "/", cleared.Path)

				var out dto.WebResponse[string]
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				require.Equal(t, "Logout successfully", out.Data)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl, app := tc.setup(t)
			req := tc.buildRequest(t, ctrl)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			require.Equal(t, tc.expectStatus, resp.StatusCode)

			if tc.assert != nil {
				tc.assert(t, resp)
			}
		})
	}
}

// failingBlacklistRepo simulates an error when adding tokens to blacklist
type failingBlacklistRepo struct{}

func (f *failingBlacklistRepo) Add(token string, tokenType constant.TokenType, duration time.Duration) error {
	return errcode.ErrRedisSet
}

func (f *failingBlacklistRepo) IsBlacklisted(token string, tokenType constant.TokenType) (bool, error) {
	return false, nil
}
