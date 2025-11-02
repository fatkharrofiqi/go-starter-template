package controller

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/http/httptest"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    miniredis "github.com/alicebob/miniredis/v2"
    "github.com/gofiber/fiber/v2"
    "github.com/redis/go-redis/v9"
    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/require"

    "go-starter-template/internal/dto"
    "go-starter-template/internal/repository"
    "go-starter-template/internal/service"
    "go-starter-template/internal/utils/errcode"
)

// setupUserController constructs a real UserController wired with sqlmock and miniredis
func setupUserController(t *testing.T) (*UserController, *fiber.App, sqlmock.Sqlmock, *miniredis.Miniredis) {
    t.Helper()

    // sqlmock for repository
    db, mock, err := sqlmock.New()
    require.NoError(t, err)

    // miniredis for RedisService
    mr, err := miniredis.Run()
    require.NoError(t, err)
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

    logger := logrus.New()
    logger.SetOutput(io.Discard)

    userRepo := repository.NewUserRepository(db)
    redisSvc := service.NewRedisService(rdb, logger)
    userSvc := service.NewUserService(userRepo, redisSvc, logger)
    ctrl := NewUserController(userSvc, logger)

    app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error {
        if code, ok := errcode.GetHTTPStatus(err); ok {
            return c.Status(code).JSON(fiber.Map{"error": err.Error()})
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
    }})
    return ctrl, app, mock, mr
}

// Table-driven tests for Me endpoint
func TestUserController_Me(t *testing.T) {
    type testcase struct {
        name         string
        setupDB      func(sqlmock.Sqlmock)
        setupAuth    func(*fiber.App)
        expectStatus int
        assert       func(*testing.T, *http.Response)
    }

    authWithUUID := func(uuid string) func(*fiber.App) {
        return func(app *fiber.App) {
            app.Use(func(c *fiber.Ctx) error {
                c.Locals("auth", &service.Claims{UUID: uuid})
                return c.Next()
            })
        }
    }

    userRow := sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
        AddRow("user-123", "Alice", "alice@example.com", "hash", time.Now(), time.Now())

    cases := []testcase{
        {
            name: "Success_DBAndCache",
            setupDB: func(mock sqlmock.Sqlmock) {
                // FindByUUID
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
                    WithArgs("user-123").
                    WillReturnRows(userRow)
                // roles (empty)
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("user-123").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                // direct permissions (empty)
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
                    WithArgs("user-123").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                // role permissions (empty)
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("user-123").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
            },
            setupAuth:    authWithUUID("user-123"),
            expectStatus: http.StatusOK,
            assert: func(t *testing.T, resp *http.Response) {
                var out dto.WebResponse[*dto.UserResponse]
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.NotNil(t, out.Data)
                require.Equal(t, "alice@example.com", out.Data.Email)
                require.Equal(t, "Alice", out.Data.Name)
                require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
            },
        },
        {
            name: "NotFound",
            setupDB: func(mock sqlmock.Sqlmock) {
                // FindByUUID returns error
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
                    WithArgs("missing").
                    WillReturnError(fmt.Errorf("no rows"))
            },
            setupAuth:    authWithUUID("missing"),
            expectStatus: http.StatusNotFound,
            assert: func(t *testing.T, resp *http.Response) {
                var out struct{ Error string `json:"error"` }
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "user not found", out.Error)
            },
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            ctrl, app, mock, mr := setupUserController(t)
            defer mr.Close()
            tc.setupAuth(app)
            app.Get("/me", ctrl.Me)

            if tc.setupDB != nil {
                tc.setupDB(mock)
            }

            req := httptest.NewRequest(http.MethodGet, "/me", nil)
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

// Table-driven tests for List endpoint
func TestUserController_List(t *testing.T) {
    type testcase struct {
        name         string
        query        string
        setupDB      func(sqlmock.Sqlmock)
        expectStatus int
        assert       func(*testing.T, *http.Response)
    }

    cases := []testcase{
        {
            name:  "Success_WithPaging",
            query: "?page=2&size=2&name=A",
            setupDB: func(mock sqlmock.Sqlmock) {
                // Count
                mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE name ILIKE $1")).
                    WithArgs("%A%").
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
                // Data
                mock.ExpectQuery(regexp.QuoteMeta("SELECT uuid, name, email, created_at, updated_at FROM users WHERE name ILIKE $1 ORDER BY created_at DESC OFFSET $2 LIMIT $3")).
                    WithArgs("%A%", 2, 2).
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "created_at", "updated_at"}).
                        AddRow("u1", "A", "a@example.com", time.Now(), time.Now()).
                        AddRow("u2", "B", "b@example.com", time.Now(), time.Now()))
            },
            expectStatus: http.StatusOK,
            assert: func(t *testing.T, resp *http.Response) {
                var out dto.WebResponse[[]*dto.UserResponse]
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Len(t, out.Data, 2)
                require.NotNil(t, out.Paging)
                require.Equal(t, 2, out.Paging.Page)
                require.Equal(t, 2, out.Paging.Size)
                require.Equal(t, int64(5), out.Paging.TotalItem)
                require.Equal(t, int64(3), out.Paging.TotalPage)
            },
        },
        {
            name:  "DefaultPaging",
            query: "",
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users ")).
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
                mock.ExpectQuery(regexp.QuoteMeta("SELECT uuid, name, email, created_at, updated_at FROM users  ORDER BY created_at DESC OFFSET $1 LIMIT $2")).
                    WithArgs(0, 10).
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "created_at", "updated_at"}).
                        AddRow("u1", "A", "a@example.com", time.Now(), time.Now()).
                        AddRow("u2", "B", "b@example.com", time.Now(), time.Now()))
            },
            expectStatus: http.StatusOK,
        },
        {
            name:  "SearchError",
            query: "?page=1&size=10",
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users ")).
                    WillReturnError(fmt.Errorf("db error"))
            },
            expectStatus: http.StatusNotFound,
            assert: func(t *testing.T, resp *http.Response) {
                var out struct{ Error string `json:"error"` }
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "failed to retrieve users", out.Error)
            },
        },
        {
            name:  "BadRequest_QueryParse",
            query: "?page=abc&size=10",
            setupDB: func(sqlmock.Sqlmock) {},
            expectStatus: http.StatusBadRequest,
            assert: func(t *testing.T, resp *http.Response) {
                var out struct{ Error string `json:"error"` }
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "bad request", out.Error)
            },
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            ctrl, app, mock, mr := setupUserController(t)
            defer mr.Close()
            app.Get("/users", ctrl.List)

            if tc.setupDB != nil {
                tc.setupDB(mock)
            }

            req := httptest.NewRequest(http.MethodGet, "/users"+tc.query, nil)
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

// Table-driven tests for Create endpoint
func TestUserController_Create(t *testing.T) {
    type testcase struct {
        name         string
        body         string
        setupDB      func(sqlmock.Sqlmock)
        expectStatus int
        assert       func(*testing.T, *http.Response)
    }

    cases := []testcase{
        {
            name: "Success",
            body: `{"name":"Alice","email":"alice@example.com","password":"secret123"}`,
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
                    WithArgs("alice@example.com").
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
                mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (uuid, name, email, password, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW())")).
                    WithArgs(sqlmock.AnyArg(), "Alice", "alice@example.com", sqlmock.AnyArg()).
                    WillReturnResult(sqlmock.NewResult(1, 1))
            },
            expectStatus: http.StatusOK,
            assert: func(t *testing.T, resp *http.Response) {
                var out dto.WebResponse[*dto.UserResponse]
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "alice@example.com", out.Data.Email)
            },
        },
        {
            name: "BadRequest_BodyParseError",
            body: `{"name":1}`,
            setupDB: func(sqlmock.Sqlmock) {},
            expectStatus: http.StatusBadRequest,
        },
        {
            name: "EmailExists",
            body: `{"name":"Alice","email":"alice@example.com","password":"secret123"}`,
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
                    WithArgs("alice@example.com").
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
            },
            expectStatus: http.StatusConflict,
            assert: func(t *testing.T, resp *http.Response) {
                var out struct{ Error string `json:"error"` }
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "user already exists", out.Error)
            },
        },
        {
            name: "InternalError_CountQuery",
            body: `{"name":"Alice","email":"alice@example.com","password":"secret123"}`,
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
                    WithArgs("alice@example.com").
                    WillReturnError(fmt.Errorf("db error"))
            },
            expectStatus: http.StatusInternalServerError,
            assert: func(t *testing.T, resp *http.Response) {
                var out struct{ Error string `json:"error"` }
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "internal server error", out.Error)
            },
        },
        {
            name: "InternalError_Insert",
            body: `{"name":"Alice","email":"alice@example.com","password":"secret123"}`,
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
                    WithArgs("alice@example.com").
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
                mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (uuid, name, email, password, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW())")).
                    WithArgs(sqlmock.AnyArg(), "Alice", "alice@example.com", sqlmock.AnyArg()).
                    WillReturnError(fmt.Errorf("insert error"))
            },
            expectStatus: http.StatusInternalServerError,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            ctrl, app, mock, mr := setupUserController(t)
            defer mr.Close()
            app.Post("/users", ctrl.Create)
            if tc.setupDB != nil {
                tc.setupDB(mock)
            }
            req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(tc.body))
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

// Table-driven tests for Update endpoint
func TestUserController_Update(t *testing.T) {
    type testcase struct {
        name         string
        uuid         string
        body         string
        setupDB      func(sqlmock.Sqlmock)
        expectStatus int
        assert       func(*testing.T, *http.Response)
    }

    newUserRow := func(uuid, name, email string) *sqlmock.Rows {
        return sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
            AddRow(uuid, name, email, "hash", time.Now(), time.Now())
    }

    cases := []testcase{
        {
            name:         "BadRequest_BodyParse",
            uuid:         "u1",
            body:         `{"name":1}`,
            setupDB:      func(sqlmock.Sqlmock) {},
            expectStatus: http.StatusBadRequest,
        },
        {
            name: "NotFound",
            uuid: "missing",
            body: `{"name":"Alice","email":"alice@example.com"}`,
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
                    WithArgs("missing").
                    WillReturnError(fmt.Errorf("no rows"))
            },
            expectStatus: http.StatusNotFound,
            assert: func(t *testing.T, resp *http.Response) {
                var out struct{ Error string `json:"error"` }
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "user not found", out.Error)
            },
        },
        {
            name: "Success_NoEmailChange",
            uuid: "u1",
            body: `{"name":"Alice","email":"old@example.com"}`,
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
                    WithArgs("u1").
                    WillReturnRows(newUserRow("u1", "OldName", "old@example.com"))
                // roles (empty)
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                // direct permissions (empty)
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                // role permissions (empty)
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET name = $1, email = $2, updated_at = NOW() WHERE uuid = $3")).
                    WithArgs("Alice", "old@example.com", "u1").
                    WillReturnResult(sqlmock.NewResult(1, 1))
            },
            expectStatus: http.StatusOK,
            assert: func(t *testing.T, resp *http.Response) {
                var out dto.WebResponse[*dto.UserResponse]
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "u1", out.Data.UUID)
                require.Equal(t, "Alice", out.Data.Name)
            },
        },
        {
            name: "EmailChanged_ExistsConflict",
            uuid: "u1",
            body: `{"name":"Alice","email":"new@example.com"}`,
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
                    WithArgs("u1").
                    WillReturnRows(newUserRow("u1", "OldName", "old@example.com"))
                // roles/permissions queries
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
                    WithArgs("new@example.com").
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
            },
            expectStatus: http.StatusConflict,
        },
        {
            name:         "BadRequest_MissingUUID",
            uuid:         "",
            body:         `{"name":"Alice","email":"alice@example.com"}`,
            setupDB:      func(sqlmock.Sqlmock) {},
            expectStatus: http.StatusBadRequest,
            assert: func(t *testing.T, resp *http.Response) {
                var out struct{ Error string `json:"error"` }
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "bad request", out.Error)
            },
        },
        {
            name: "EmailChanged_InternalErrorOnCount",
            uuid: "u1",
            body: `{"name":"Alice","email":"new@example.com"}`,
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
                    WithArgs("u1").
                    WillReturnRows(newUserRow("u1", "OldName", "old@example.com"))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE email = $1")).
                    WithArgs("new@example.com").
                    WillReturnError(fmt.Errorf("count error"))
            },
            expectStatus: http.StatusInternalServerError,
        },
        {
            name: "InternalError_UpdateExec",
            uuid: "u1",
            body: `{"name":"Alice","email":"old@example.com"}`,
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
                    WithArgs("u1").
                    WillReturnRows(newUserRow("u1", "OldName", "old@example.com"))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET name = $1, email = $2, updated_at = NOW() WHERE uuid = $3")).
                    WithArgs("Alice", "old@example.com", "u1").
                    WillReturnError(fmt.Errorf("update error"))
            },
            expectStatus: http.StatusInternalServerError,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            ctrl, app, mock, mr := setupUserController(t)
            defer mr.Close()
            // Register both routes to cover missing UUID branch
            app.Put("/users/:uuid", ctrl.Update)
            app.Put("/users", ctrl.Update)

            if tc.setupDB != nil {
                tc.setupDB(mock)
            }

            path := "/users/" + tc.uuid
            if tc.uuid == "" {
                path = "/users"
            }
            req := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(tc.body))
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

// Table-driven tests for Delete endpoint
func TestUserController_Delete(t *testing.T) {
    type testcase struct {
        name         string
        uuid         string
        setupDB      func(sqlmock.Sqlmock)
        expectStatus int
        assert       func(*testing.T, *http.Response)
    }

    cases := []testcase{
        {
            name: "NotFound",
            uuid: "missing",
            setupDB: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
                    WithArgs("missing").
                    WillReturnError(fmt.Errorf("no rows"))
            },
            expectStatus: http.StatusNotFound,
        },
        {
            name: "BadRequest_MissingUUID",
            uuid: "",
            setupDB: func(sqlmock.Sqlmock) {},
            expectStatus: http.StatusBadRequest,
            assert: func(t *testing.T, resp *http.Response) {
                var out struct{ Error string `json:"error"` }
                require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
                require.Equal(t, "bad request", out.Error)
            },
        },
        {
            name: "Success",
            uuid: "u1",
            setupDB: func(mock sqlmock.Sqlmock) {
                userRow := sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                    AddRow("u1", "Name", "email@example.com", "hash", time.Now(), time.Now())
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
                    WithArgs("u1").
                    WillReturnRows(userRow)
                // roles/permissions queries
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectExec(regexp.QuoteMeta("DELETE FROM users WHERE uuid = $1")).
                    WithArgs("u1").
                    WillReturnResult(sqlmock.NewResult(1, 1))
            },
            expectStatus: http.StatusNoContent,
        },
        {
            name: "InternalError_DeleteExec",
            uuid: "u1",
            setupDB: func(mock sqlmock.Sqlmock) {
                userRow := sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                    AddRow("u1", "Name", "email@example.com", "hash", time.Now(), time.Now())
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`)).
                    WithArgs("u1").
                    WillReturnRows(userRow)
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectQuery(regexp.QuoteMeta(`SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1`)).
                    WithArgs("u1").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}))
                mock.ExpectExec(regexp.QuoteMeta("DELETE FROM users WHERE uuid = $1")).
                    WithArgs("u1").
                    WillReturnError(fmt.Errorf("delete error"))
            },
            expectStatus: http.StatusInternalServerError,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            ctrl, app, mock, mr := setupUserController(t)
            defer mr.Close()
            app.Delete("/users/:uuid", ctrl.Delete)
            app.Delete("/users", ctrl.Delete)

            if tc.setupDB != nil {
                tc.setupDB(mock)
            }

            path := "/users/" + tc.uuid
            if tc.uuid == "" {
                path = "/users"
            }
            req := httptest.NewRequest(http.MethodDelete, path, nil)
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