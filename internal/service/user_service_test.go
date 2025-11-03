package service

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"go-starter-template/internal/dto"
	"go-starter-template/internal/repository"
	"go-starter-template/internal/utils/errcode"
)

// setupRepoAndUow replicates the helper in auth_service_test.go to produce a sqlmock-backed repository.
func setupRepo(t *testing.T) (*repository.UserRepository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	repo := repository.NewUserRepository(db)
	cleanup := func() { _ = db.Close() }
	return repo, mock, cleanup
}

// fakeRedisClient satisfies redisClient for testing cache behavior in GetUser.
type userTestRedisClient struct {
	getFunc func(ctx context.Context, key string) *redis.StringCmd
	setFunc func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

func (f *userTestRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return f.getFunc(ctx, key)
}
func (f *userTestRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return f.setFunc(ctx, key, value, expiration)
}

func TestUserService_GetUser(t *testing.T) {
	logger := silentLogger()

	type testcase struct {
		name      string
		setupDB   func(sqlmock.Sqlmock)
		setupRds  func() redisClient
		uuid      string
		expectErr error
		assert    func(t *testing.T, resp string)
	}

	cases := []testcase{
		{
			name:    "CacheHit",
			uuid:    "user-123",
			setupDB: func(sqlmock.Sqlmock) {},
			setupRds: func() redisClient {
				return &userTestRedisClient{
					getFunc: func(ctx context.Context, key string) *redis.StringCmd {
						cmd := redis.NewStringCmd(ctx)
						cmd.SetVal("{\"data\":{\"email\":\"alice@example.com\"}}")
						return cmd
					},
					setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
						return redis.NewStatusCmd(ctx)
					},
				}
			},
			expectErr: nil,
			assert: func(t *testing.T, resp string) {
				require.Contains(t, resp, "alice@example.com")
			},
		},
		{
			name: "CacheMiss_DBNotFound",
			uuid: "missing",
			setupDB: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("missing").
					WillReturnError(errors.New("no rows"))
			},
			setupRds: func() redisClient {
				return &userTestRedisClient{
					getFunc: func(ctx context.Context, key string) *redis.StringCmd {
						cmd := redis.NewStringCmd(ctx)
						cmd.SetErr(redis.Nil)
						return cmd
					},
					setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
						return redis.NewStatusCmd(ctx)
					},
				}
			},
			expectErr: errcode.ErrUserNotFound,
			assert: func(t *testing.T, resp string) {
				require.Empty(t, resp)
			},
		},
		{
			name: "CacheMiss_DBFound_StoresCache",
			uuid: "user-123",
			setupDB: func(m sqlmock.Sqlmock) {
				userRow := sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
					AddRow("user-123", "Alice", "alice@example.com", "hash", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("user-123").
					WillReturnRows(userRow)
				m.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("user-123").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
					WithArgs("user-123").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("user-123").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
			},
			setupRds: func() redisClient {
				return &userTestRedisClient{
					getFunc: func(ctx context.Context, key string) *redis.StringCmd {
						cmd := redis.NewStringCmd(ctx)
						cmd.SetErr(redis.Nil)
						return cmd
					},
					setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
						cmd := redis.NewStatusCmd(ctx)
						cmd.SetVal("OK")
						return cmd
					},
				}
			},
			expectErr: nil,
			assert: func(t *testing.T, resp string) {
				require.Contains(t, resp, "alice@example.com")
			},
		},
		{
			name: "CacheMiss_DBFound_SetError",
			uuid: "user-123",
			setupDB: func(m sqlmock.Sqlmock) {
				userRow := sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
					AddRow("user-123", "Alice", "alice@example.com", "hash", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("user-123").
					WillReturnRows(userRow)
				m.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("user-123").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
					WithArgs("user-123").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("user-123").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
			},
			setupRds: func() redisClient {
				return &userTestRedisClient{
					getFunc: func(ctx context.Context, key string) *redis.StringCmd {
						cmd := redis.NewStringCmd(ctx)
						cmd.SetErr(redis.Nil)
						return cmd
					},
					setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
						cmd := redis.NewStatusCmd(ctx)
						cmd.SetErr(errors.New("redis set error"))
						return cmd
					},
				}
			},
			expectErr: errors.New("redis set error"),
			assert: func(t *testing.T, resp string) {
				require.Empty(t, resp)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock, cleanup := setupRepo(t)
			defer cleanup()
			require.NotNil(t, repo)
			if tc.setupDB != nil {
				tc.setupDB(mock)
			}
			redisSvc := NewRedisService(tc.setupRds(), logger)
			svc := NewUserService(repo, redisSvc, logger)
			resp, err := svc.GetUser(context.Background(), tc.uuid)
			if e := mock.ExpectationsWereMet(); e != nil {
				t.Logf("sqlmock expectations error: %v", e)
			}
			if tc.expectErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectErr, err)
			} else {
				require.NoError(t, err)
			}
			if tc.assert != nil {
				tc.assert(t, resp)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserService_Search(t *testing.T) {
	logger := silentLogger()
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	svc := NewUserService(repo, NewRedisService(&userTestRedisClient{}, logger), logger)

	type testcase struct {
		name      string
		request   *dto.SearchUserRequest
		setupDB   func(sqlmock.Sqlmock)
		expectErr error
		assert    func(t *testing.T, users []*dto.UserResponse, total int64)
	}

	mkRows := func(n int) *sqlmock.Rows {
		rows := sqlmock.NewRows([]string{"uuid", "name", "email", "created_at", "updated_at"})
		for i := 0; i < n; i++ {
			rows.AddRow("u"+string(rune('1'+i)), "N"+string(rune('1'+i)), "e"+string(rune('1'+i))+"@ex.com", time.Now(), time.Now())
		}
		return rows
	}

	cases := []testcase{
		{
			name:    "DefaultPaging_Success",
			request: &dto.SearchUserRequest{},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users ")).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
				m.ExpectQuery(regexp.QuoteMeta("SELECT uuid, name, email, created_at, updated_at FROM users  ORDER BY created_at DESC OFFSET $1 LIMIT $2")).
					WithArgs(0, 10).
					WillReturnRows(mkRows(2))
			},
			expectErr: nil,
			assert: func(t *testing.T, users []*dto.UserResponse, total int64) {
				require.Len(t, users, 2)
				require.Equal(t, int64(2), total)
			},
		},
		{
			name:    "FilterByNameAndEmail_Success",
			request: &dto.SearchUserRequest{Name: "Al", Email: "ex"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE name ILIKE $1 AND email ILIKE $2")).
					WithArgs("%Al%", "%ex%").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				m.ExpectQuery(regexp.QuoteMeta("SELECT uuid, name, email, created_at, updated_at FROM users WHERE name ILIKE $1 AND email ILIKE $2 ORDER BY created_at DESC OFFSET $3 LIMIT $4")).
					WithArgs("%Al%", "%ex%", 0, 10).
					WillReturnRows(mkRows(1))
			},
			assert: func(t *testing.T, users []*dto.UserResponse, total int64) {
				require.Len(t, users, 1)
				require.Equal(t, int64(1), total)
			},
		},
		{
			name:    "SearchError_CountQuery",
			request: &dto.SearchUserRequest{Page: 1, Size: 10},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users ")).
					WillReturnError(errors.New("db error"))
			},
			expectErr: errcode.ErrUserSearchFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupDB != nil {
				tc.setupDB(mock)
			}
			users, total, err := svc.Search(context.Background(), tc.request)
			if tc.expectErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectErr, err)
			} else {
				require.NoError(t, err)
			}
			if tc.assert != nil {
				tc.assert(t, users, total)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	logger := silentLogger()
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()
	svc := NewUserService(repo, NewRedisService(&userTestRedisClient{}, logger), logger)

	type testcase struct {
		name      string
		uuid      string
		req       *dto.UpdateUserRequest
		setupDB   func(sqlmock.Sqlmock)
		expectErr error
		assert    func(t *testing.T, resp *dto.UserResponse)
	}

	newUserRow := func(uuid, name, email string) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
			AddRow(uuid, name, email, "hash", time.Now(), time.Now())
	}

	cases := []testcase{
		{
			name: "NotFound",
			uuid: "missing",
			req:  &dto.UpdateUserRequest{Name: "Alice", Email: "alice@example.com"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("missing").
					WillReturnError(errors.New("no rows"))
			},
			expectErr: errcode.ErrUserNotFound,
		},
		{
			name: "EmailUnchanged_UpdateSuccess",
			uuid: "u1",
			req:  &dto.UpdateUserRequest{Name: "Alice", Email: "old@example.com"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("u1").
					WillReturnRows(newUserRow("u1", "Old", "old@example.com"))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectExec(regexp.QuoteMeta("UPDATE users SET name = $1, email = $2, updated_at = NOW() WHERE uuid = $3")).
					WithArgs("Alice", "old@example.com", "u1").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			assert: func(t *testing.T, resp *dto.UserResponse) {
				require.NotNil(t, resp)
				require.Equal(t, "u1", resp.UUID)
				require.Equal(t, "Alice", resp.Name)
			},
		},
		{
			name: "EmailChanged_ExistsConflict",
			uuid: "u1",
			req:  &dto.UpdateUserRequest{Name: "Alice", Email: "new@example.com"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("u1").
					WillReturnRows(newUserRow("u1", "Old", "old@example.com"))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
					WithArgs("new@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectErr: errcode.ErrUserAlreadyExists,
		},
		{
			name: "EmailChanged_InternalErrorOnCount",
			uuid: "u1",
			req:  &dto.UpdateUserRequest{Name: "Alice", Email: "new@example.com"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("u1").
					WillReturnRows(newUserRow("u1", "Old", "old@example.com"))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
					WithArgs("new@example.com").
					WillReturnError(errors.New("count error"))
			},
			expectErr: errcode.ErrInternalServerError,
		},
		{
			name: "UpdateExecError",
			uuid: "u1",
			req:  &dto.UpdateUserRequest{Name: "Alice", Email: "old@example.com"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("u1").
					WillReturnRows(newUserRow("u1", "Old", "old@example.com"))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectExec(regexp.QuoteMeta("UPDATE users SET name = $1, email = $2, updated_at = NOW() WHERE uuid = $3")).
					WithArgs("Alice", "old@example.com", "u1").
					WillReturnError(errors.New("update error"))
			},
			expectErr: errcode.ErrInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupDB != nil {
				tc.setupDB(mock)
			}
			resp, err := svc.UpdateUser(context.Background(), tc.uuid, tc.req)
			if tc.expectErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectErr, err)
			} else {
				require.NoError(t, err)
			}
			if tc.assert != nil {
				tc.assert(t, resp)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	logger := silentLogger()
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()
	svc := NewUserService(repo, NewRedisService(&userTestRedisClient{}, logger), logger)

	type testcase struct {
		name      string
		uuid      string
		setupDB   func(sqlmock.Sqlmock)
		expectErr error
	}

	cases := []testcase{
		{
			name: "NotFound",
			uuid: "missing",
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("missing").
					WillReturnError(errors.New("no rows"))
			},
			expectErr: errcode.ErrUserNotFound,
		},
		{
			name: "DeleteSuccess",
			uuid: "u1",
			setupDB: func(m sqlmock.Sqlmock) {
				userRow := sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
					AddRow("u1", "Name", "e@example.com", "hash", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("u1").
					WillReturnRows(userRow)
				m.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectExec(regexp.QuoteMeta("DELETE FROM users WHERE uuid = $1")).
					WithArgs("u1").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "DeleteExecError",
			uuid: "u1",
			setupDB: func(m sqlmock.Sqlmock) {
				userRow := sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
					AddRow("u1", "Name", "e@example.com", "hash", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
					WithArgs("u1").
					WillReturnRows(userRow)
				m.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
					WithArgs("u1").
					WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
				m.ExpectExec(regexp.QuoteMeta("DELETE FROM users WHERE uuid = $1")).
					WithArgs("u1").
					WillReturnError(errors.New("delete error"))
			},
			expectErr: errcode.ErrInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupDB != nil {
				tc.setupDB(mock)
			}
			err := svc.DeleteUser(context.Background(), tc.uuid)
			if tc.expectErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectErr, err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
func TestUserService_CreateUser(t *testing.T) {
	logger := silentLogger()
	repo, mock, cleanup := setupRepo(t)
	defer cleanup()

	type testcase struct {
		name      string
		req       *dto.CreateUserRequest
		setupDB   func(sqlmock.Sqlmock)
		mutateSvc func(*UserService)
		expectErr error
		assert    func(*testing.T, *dto.UserResponse)
	}

	cases := []testcase{
		{
			name: "CountError",
			req:  &dto.CreateUserRequest{Name: "Alice", Email: "alice@example.com", Password: "pass"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
					WithArgs("alice@example.com").
					WillReturnError(errors.New("db error"))
			},
			expectErr: errcode.ErrInternalServerError,
		},
		{
			name: "EmailExists_Conflict",
			req:  &dto.CreateUserRequest{Name: "Alice", Email: "alice@example.com", Password: "pass"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
					WithArgs("alice@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectErr: errcode.ErrUserAlreadyExists,
		},
		{
			name: "HashError",
			req:  &dto.CreateUserRequest{Name: "Alice", Email: "alice@example.com", Password: "pass"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
					WithArgs("alice@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			mutateSvc: func(s *UserService) {
				s.hashPassword = func(_ []byte, _ int) ([]byte, error) { return nil, errors.New("hash error") }
			},
			expectErr: errcode.ErrPasswordEncryption,
		},
		{
			name: "CreateExecError",
			req:  &dto.CreateUserRequest{Name: "Alice", Email: "alice@example.com", Password: "pass"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
					WithArgs("alice@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				m.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO users (uuid, name, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
    `)).
					WithArgs(sqlmock.AnyArg(), "Alice", "alice@example.com", sqlmock.AnyArg()).
					WillReturnError(errors.New("create error"))
			},
			expectErr: errcode.ErrInternalServerError,
		},
		{
			name: "CreateSuccess",
			req:  &dto.CreateUserRequest{Name: "Alice", Email: "alice@example.com", Password: "pass"},
			setupDB: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
					WithArgs("alice@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				m.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO users (uuid, name, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
    `)).
					WithArgs(sqlmock.AnyArg(), "Alice", "alice@example.com", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			assert: func(t *testing.T, resp *dto.UserResponse) {
				require.NotNil(t, resp)
				require.Equal(t, "Alice", resp.Name)
				require.Equal(t, "alice@example.com", resp.Email)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupDB != nil {
				tc.setupDB(mock)
			}
			// fresh service per test to avoid state leakage from mutateSvc
			svc := NewUserService(repo, NewRedisService(&userTestRedisClient{}, logger), logger)
			if tc.mutateSvc != nil {
				tc.mutateSvc(svc)
			}
			resp, err := svc.CreateUser(context.Background(), tc.req)
			if tc.expectErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectErr, err)
			} else {
				require.NoError(t, err)
			}
			if tc.assert != nil {
				tc.assert(t, resp)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
