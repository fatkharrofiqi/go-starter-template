package repository

import (
    "context"
    "database/sql"
    "errors"
    "regexp"
    "testing"
    "time"

    "go-starter-template/internal/dto"
    "go-starter-template/internal/model"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/stretchr/testify/require"
)

func TestUserRepository_CountByEmail(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    repo := NewUserRepository(db)

    type tc struct {
        name      string
        setupMock func()
        email     string
        assert    func(t *testing.T, total int64, err error)
    }

    query := `SELECT COUNT(*) FROM users WHERE email = $1`

    cases := []tc{
        {
            name: "Success",
            email: "a@example.com",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(query)).
                    WithArgs("a@example.com").
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
            },
            assert: func(t *testing.T, total int64, err error) {
                require.NoError(t, err)
                require.Equal(t, int64(2), total)
            },
        },
        {
            name: "QueryError",
            email: "b@example.com",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(query)).
                    WithArgs("b@example.com").
                    WillReturnError(errors.New("db error"))
            },
            assert: func(t *testing.T, total int64, err error) {
                require.Error(t, err)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            c.setupMock()
            got, err := repo.CountByEmail(context.Background(), c.email)
            c.assert(t, got, err)
            require.NoError(t, mock.ExpectationsWereMet())
        })
    }
}

func TestUserRepository_FindByEmail(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    repo := NewUserRepository(db)

    type tc struct {
        name      string
        setupMock func()
        email     string
        assert    func(t *testing.T, u *model.User, err error)
    }

    query := `SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE email = $1 LIMIT 1`
    now := time.Now()

    cases := []tc{
        {
            name: "Success",
            email: "john@example.com",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(query)).
                    WithArgs("john@example.com").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                        AddRow("u1", "John", "john@example.com", "pass", now, now))
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.NoError(t, err)
                require.Equal(t, "u1", u.UUID)
                require.Equal(t, "John", u.Name)
            },
        },
        {
            name: "NotFound",
            email: "missing@example.com",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(query)).
                    WithArgs("missing@example.com").
                    WillReturnError(sql.ErrNoRows)
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.Error(t, err)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            var u model.User
            c.setupMock()
            err := repo.FindByEmail(context.Background(), &u, c.email)
            c.assert(t, &u, err)
            require.NoError(t, mock.ExpectationsWereMet())
        })
    }
}

func TestUserRepository_FindByUUID(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    repo := NewUserRepository(db)

    type tc struct {
        name      string
        setupMock func()
        uuid      string
        assert    func(t *testing.T, u *model.User, err error)
    }

    userQuery := `SELECT uuid, name, email, password, created_at, updated_at FROM users WHERE uuid = $1 LIMIT 1`
    rolesQuery := `
        SELECT r.uuid, r.name
        FROM roles r
        INNER JOIN user_roles ur ON ur.role_uuid = r.uuid
        WHERE ur.user_uuid = $1
    `
    directPermQuery := `
        SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN user_permissions up ON up.permission_uuid = p.uuid
        WHERE up.user_uuid = $1
    `
    rolePermQuery := `
        SELECT p.uuid, p.name
        FROM permissions p
        INNER JOIN role_permissions rp ON rp.permission_uuid = p.uuid
        INNER JOIN user_roles ur ON ur.role_uuid = rp.role_uuid
        WHERE ur.user_uuid = $1
    `

    now := time.Now()

    cases := []tc{
        {
            name: "Success",
            uuid: "u2",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(userQuery)).
                    WithArgs("u2").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                        AddRow("u2", "Jane", "jane@example.com", "pass", now, now))

                mock.ExpectQuery(regexp.QuoteMeta(rolesQuery)).
                    WithArgs("u2").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow("r1", "Admin").AddRow("r2", "User"))

                mock.ExpectQuery(regexp.QuoteMeta(directPermQuery)).
                    WithArgs("u2").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow("p1", "read"))

                mock.ExpectQuery(regexp.QuoteMeta(rolePermQuery)).
                    WithArgs("u2").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow("p2", "write"))
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.NoError(t, err)
                require.Equal(t, "u2", u.UUID)
                require.Len(t, u.Roles, 2)
                require.Len(t, u.Permissions, 2)
            },
        },
        {
            name: "UserNotFound",
            uuid: "u404",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(userQuery)).
                    WithArgs("u404").
                    WillReturnError(sql.ErrNoRows)
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.Error(t, err)
            },
        },
        {
            name: "UserScanError",
            uuid: "uBad",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(userQuery)).
                    WithArgs("uBad").
                    // invalid time values to force scan error on created_at/updated_at
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                        AddRow("uBad", "Bad", "bad@example.com", "pass", "bad-time", "bad-time"))
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.Error(t, err)
            },
        },
        {
            name: "RolesQueryError",
            uuid: "u3",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(userQuery)).
                    WithArgs("u3").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                        AddRow("u3", "Jim", "jim@example.com", "pass", now, now))
                mock.ExpectQuery(regexp.QuoteMeta(rolesQuery)).
                    WithArgs("u3").
                    WillReturnError(errors.New("roles query error"))
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.Error(t, err)
            },
        },
        {
            name: "DirectPermScanError",
            uuid: "u4",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(userQuery)).
                    WithArgs("u4").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                        AddRow("u4", "Jill", "jill@example.com", "pass", now, now))
                mock.ExpectQuery(regexp.QuoteMeta(rolesQuery)).
                    WithArgs("u4").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow("r1", "Admin"))
                mock.ExpectQuery(regexp.QuoteMeta(directPermQuery)).
                    WithArgs("u4").
                    // NULL value to force scan error when scanning into string fields
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow(nil, "read"))
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.Error(t, err)
            },
        },
        {
            name: "RolesScanError",
            uuid: "u2a",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(userQuery)).
                    WithArgs("u2a").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                        AddRow("u2a", "Jane", "jane@example.com", "pass", now, now))

                mock.ExpectQuery(regexp.QuoteMeta(rolesQuery)).
                    WithArgs("u2a").
                    // NULL value to force scan error when scanning into string fields
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow(nil, "Admin"))
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.Error(t, err)
            },
        },
        {
            name: "DirectPermQueryError",
            uuid: "u5",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(userQuery)).
                    WithArgs("u5").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                        AddRow("u5", "Jack", "jack@example.com", "pass", now, now))
                mock.ExpectQuery(regexp.QuoteMeta(rolesQuery)).
                    WithArgs("u5").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow("r1", "Admin"))
                mock.ExpectQuery(regexp.QuoteMeta(directPermQuery)).
                    WithArgs("u5").
                    WillReturnError(errors.New("direct perm query error"))
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.Error(t, err)
            },
        },
        {
            name: "RolePermQueryError",
            uuid: "u6",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(userQuery)).
                    WithArgs("u6").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                        AddRow("u6", "Jenny", "jenny@example.com", "pass", now, now))
                mock.ExpectQuery(regexp.QuoteMeta(rolesQuery)).
                    WithArgs("u6").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow("r1", "Admin"))
                mock.ExpectQuery(regexp.QuoteMeta(directPermQuery)).
                    WithArgs("u6").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow("p1", "read"))
                mock.ExpectQuery(regexp.QuoteMeta(rolePermQuery)).
                    WithArgs("u6").
                    WillReturnError(errors.New("role perm query error"))
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.Error(t, err)
            },
        },
        {
            name: "RolePermScanError",
            uuid: "u7",
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(userQuery)).
                    WithArgs("u7").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "password", "created_at", "updated_at"}).
                        AddRow("u7", "Julia", "julia@example.com", "pass", now, now))
                mock.ExpectQuery(regexp.QuoteMeta(rolesQuery)).
                    WithArgs("u7").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow("r1", "Admin"))
                mock.ExpectQuery(regexp.QuoteMeta(directPermQuery)).
                    WithArgs("u7").
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow("p1", "read"))
                mock.ExpectQuery(regexp.QuoteMeta(rolePermQuery)).
                    WithArgs("u7").
                    // NULL values to force scan error into *string fields
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name"}).AddRow(nil, "write"))
            },
            assert: func(t *testing.T, u *model.User, err error) {
                require.Error(t, err)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            var u model.User
            c.setupMock()
            err := repo.FindByUUID(context.Background(), &u, c.uuid)
            c.assert(t, &u, err)
            require.NoError(t, mock.ExpectationsWereMet())
        })
    }
}

func TestUserRepository_Search(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    repo := NewUserRepository(db)
    now := time.Now()

    type tc struct {
        name      string
        setupMock func()
        req       *dto.SearchUserRequest
        assert    func(t *testing.T, users []*model.User, total int64, err error)
    }

    countBase := "SELECT COUNT(*) FROM users "
    dataBase := "SELECT uuid, name, email, created_at, updated_at FROM users "

    cases := []tc{
        {
            name: "NoFilters_Success",
            req:  &dto.SearchUserRequest{Page: 1, Size: 2},
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(countBase)).
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
                mock.ExpectQuery(regexp.QuoteMeta(dataBase + " ORDER BY created_at DESC OFFSET $1 LIMIT $2")).
                    WithArgs(0, 2).
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "created_at", "updated_at"}).
                        AddRow("u1", "A", "a@example.com", now, now).
                        AddRow("u2", "B", "b@example.com", now, now))
            },
            assert: func(t *testing.T, users []*model.User, total int64, err error) {
                require.NoError(t, err)
                require.Equal(t, int64(2), total)
                require.Len(t, users, 2)
            },
        },
        {
            name: "WithFilters_Success",
            req:  &dto.SearchUserRequest{Name: "Al", Email: "ex", Page: 2, Size: 5},
            setupMock: func() {
                where := "WHERE name ILIKE $1 AND email ILIKE $2"
                mock.ExpectQuery(regexp.QuoteMeta(countBase + where)).
                    WithArgs("%Al%", "%ex%").
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
                mock.ExpectQuery(regexp.QuoteMeta(dataBase + where + " ORDER BY created_at DESC OFFSET $3 LIMIT $4")).
                    WithArgs("%Al%", "%ex%", 5, 5).
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "created_at", "updated_at"}).
                        AddRow("u1", "Alison", "alison@example.com", now, now).
                        AddRow("u2", "Alex", "alex@example.com", now, now))
            },
            assert: func(t *testing.T, users []*model.User, total int64, err error) {
                require.NoError(t, err)
                require.Equal(t, int64(7), total)
                require.Len(t, users, 2)
            },
        },
        {
            name: "CountError",
            req:  &dto.SearchUserRequest{Page: 1, Size: 1},
            setupMock: func() {
                mock.ExpectQuery(regexp.QuoteMeta(countBase)).
                    WillReturnError(errors.New("count failed"))
            },
            assert: func(t *testing.T, users []*model.User, total int64, err error) {
                require.Error(t, err)
            },
        },
        {
            name: "QueryError",
            req:  &dto.SearchUserRequest{Name: "x", Page: 1, Size: 1},
            setupMock: func() {
                where := "WHERE name ILIKE $1"
                mock.ExpectQuery(regexp.QuoteMeta(countBase + where)).
                    WithArgs("%x%").
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
                mock.ExpectQuery(regexp.QuoteMeta(dataBase + where + " ORDER BY created_at DESC OFFSET $2 LIMIT $3")).
                    WithArgs("%x%", 0, 1).
                    WillReturnError(errors.New("query failed"))
            },
            assert: func(t *testing.T, users []*model.User, total int64, err error) {
                require.Error(t, err)
            },
        },
        {
            name: "ScanError",
            req:  &dto.SearchUserRequest{Email: "x", Page: 1, Size: 1},
            setupMock: func() {
                where := "WHERE email ILIKE $1"
                mock.ExpectQuery(regexp.QuoteMeta(countBase + where)).
                    WithArgs("%x%").
                    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
                mock.ExpectQuery(regexp.QuoteMeta(dataBase + where + " ORDER BY created_at DESC OFFSET $2 LIMIT $3")).
                    WithArgs("%x%", 0, 1).
                    WillReturnRows(sqlmock.NewRows([]string{"uuid", "name", "email", "created_at", "updated_at"}).
                        // wrong types to force scan error: created_at/updated_at should be time.Time
                        AddRow("uX", "X", "x@example.com", "bad-time", "bad-time"))
            },
            assert: func(t *testing.T, users []*model.User, total int64, err error) {
                require.Error(t, err)
            },
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            c.req.SetDefault()
            c.setupMock()
            users, total, err := repo.Search(context.Background(), c.req)
            c.assert(t, users, total, err)
            require.NoError(t, mock.ExpectationsWereMet())
        })
    }
}

func TestUserRepository_Create_Update_Delete(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    repo := NewUserRepository(db)

    insertQuery := `
        INSERT INTO users (uuid, name, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
    `
    updateQuery := `
        UPDATE users SET name = $1, email = $2, updated_at = NOW() WHERE uuid = $3
    `
    deleteQuery := `DELETE FROM users WHERE uuid = $1`

    type tc struct {
        name      string
        setupMock func()
        action    func() error
        expectErr bool
    }

    u := &model.User{UUID: "u10", Name: "Ten", Email: "ten@example.com", Password: "pass"}

    cases := []tc{
        {
            name: "CreateSuccess",
            setupMock: func() {
                mock.ExpectExec(regexp.QuoteMeta(insertQuery)).
                    WithArgs("u10", "Ten", "ten@example.com", "pass").
                    WillReturnResult(sqlmock.NewResult(1, 1))
            },
            action: func() error { return repo.Create(context.Background(), u) },
            expectErr: false,
        },
        {
            name: "CreateError",
            setupMock: func() {
                mock.ExpectExec(regexp.QuoteMeta(insertQuery)).
                    WithArgs("u10", "Ten", "ten@example.com", "pass").
                    WillReturnError(errors.New("insert error"))
            },
            action: func() error { return repo.Create(context.Background(), u) },
            expectErr: true,
        },
        {
            name: "UpdateSuccess",
            setupMock: func() {
                mock.ExpectExec(regexp.QuoteMeta(updateQuery)).
                    WithArgs("Ten", "ten@example.com", "u10").
                    WillReturnResult(sqlmock.NewResult(1, 1))
            },
            action: func() error { return repo.Update(context.Background(), u) },
            expectErr: false,
        },
        {
            name: "UpdateError",
            setupMock: func() {
                mock.ExpectExec(regexp.QuoteMeta(updateQuery)).
                    WithArgs("Ten", "ten@example.com", "u10").
                    WillReturnError(errors.New("update error"))
            },
            action: func() error { return repo.Update(context.Background(), u) },
            expectErr: true,
        },
        {
            name: "DeleteSuccess",
            setupMock: func() {
                mock.ExpectExec(regexp.QuoteMeta(deleteQuery)).
                    WithArgs("u10").
                    WillReturnResult(sqlmock.NewResult(1, 1))
            },
            action: func() error { return repo.Delete(context.Background(), u) },
            expectErr: false,
        },
        {
            name: "DeleteError",
            setupMock: func() {
                mock.ExpectExec(regexp.QuoteMeta(deleteQuery)).
                    WithArgs("u10").
                    WillReturnError(errors.New("delete error"))
            },
            action: func() error { return repo.Delete(context.Background(), u) },
            expectErr: true,
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            c.setupMock()
            err := c.action()
            if c.expectErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
            require.NoError(t, mock.ExpectationsWereMet())
        })
    }
}