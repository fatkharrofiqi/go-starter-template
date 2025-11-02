package seeder

import (
    "database/sql"
    "log"
    "time"

    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    "go-starter-template/internal/model"
)

func Seed(db *sql.DB) {
    // Cleanup existing records in dependency order
    for _, table := range []string{"user_roles", "role_permissions", "user_permissions", "users", "roles", "permissions"} {
        if _, err := db.Exec("DELETE FROM " + table); err != nil {
            log.Fatalf("Failed to delete from %s: %v", table, err)
        }
    }

    // Prepare base data using model structs
    adminRole := model.Role{UUID: uuid.NewString(), Name: "admin"}
    userRole := model.Role{UUID: uuid.NewString(), Name: "user"}
    roles := []model.Role{adminRole, userRole}

    // Insert roles
    for _, r := range roles {
        if _, err := db.Exec(`INSERT INTO roles (uuid, name) VALUES ($1, $2)`, r.UUID, r.Name); err != nil {
            log.Fatalf("Failed to insert role %s: %v", r.Name, err)
        }
    }

    // Permissions
    newPerm := func(name string) model.Permission { return model.Permission{UUID: uuid.NewString(), Name: name} }
    crudUser := []model.Permission{
        newPerm("read-user"),
        newPerm("write-user"),
        newPerm("delete-user"),
        newPerm("update-user"),
    }
    crudPermissions := []model.Permission{
        newPerm("read-permission"),
        newPerm("write-permission"),
        newPerm("delete-permission"),
        newPerm("update-permission"),
    }
    crudRole := []model.Permission{
        newPerm("read-role"),
        newPerm("write-role"),
        newPerm("delete-role"),
        newPerm("update-role"),
    }
    otherPermission := newPerm("read-other")

    // Insert permissions
    insertPerm := func(p model.Permission) {
        if _, err := db.Exec(`INSERT INTO permissions (uuid, name) VALUES ($1, $2)`, p.UUID, p.Name); err != nil {
            log.Fatalf("Failed to insert permission %s: %v", p.Name, err)
        }
    }
    for _, p := range crudUser { insertPerm(p) }
    for _, p := range crudPermissions { insertPerm(p) }
    for _, p := range crudRole { insertPerm(p) }
    insertPerm(otherPermission)

    // Create a test user
    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
    now := time.Now()
    user := model.User{
        UUID:      uuid.NewString(),
        Name:      "Test",
        Email:     "test@test.com",
        Password:  string(hashedPassword),
        CreatedAt: now,
        UpdatedAt: now,
    }
    if _, err := db.Exec(`INSERT INTO users (uuid, name, email, password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
        user.UUID, user.Name, user.Email, user.Password, user.CreatedAt, user.UpdatedAt); err != nil {
        log.Fatalf("Failed to insert test user: %v", err)
    }

    // Assign permissions to admin role
    for _, p := range append(crudPermissions, crudRole...) {
        if _, err := db.Exec(`INSERT INTO role_permissions (role_uuid, permission_uuid) VALUES ($1, $2)`, adminRole.UUID, p.UUID); err != nil {
            log.Fatalf("Failed to assign permission %s to admin role: %v", p.Name, err)
        }
    }

    // Assign permissions to user role
    for _, p := range crudUser {
        if _, err := db.Exec(`INSERT INTO role_permissions (role_uuid, permission_uuid) VALUES ($1, $2)`, userRole.UUID, p.UUID); err != nil {
            log.Fatalf("Failed to assign permission %s to user role: %v", p.Name, err)
        }
    }

    // Assign roles to user
    for _, r := range []model.Role{adminRole, userRole} {
        if _, err := db.Exec(`INSERT INTO user_roles (user_uuid, role_uuid) VALUES ($1, $2)`, user.UUID, r.UUID); err != nil {
            log.Fatalf("Failed to assign role %s to user: %v", r.Name, err)
        }
    }

    // Assign extra permission to user
    if _, err := db.Exec(`INSERT INTO user_permissions (user_uuid, permission_uuid) VALUES ($1, $2)`, user.UUID, otherPermission.UUID); err != nil {
        log.Fatalf("Failed to assign other permission to user: %v", err)
    }
}
