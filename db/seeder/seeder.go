package seeder

import (
	"go-starter-template/internal/model"
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) {
	// Delete existing records
	if err := db.Exec("DELETE FROM user_roles").Error; err != nil {
		log.Fatalf("Failed to delete user_roles: %v", err)
	}
	if err := db.Exec("DELETE FROM role_permissions").Error; err != nil {
		log.Fatalf("Failed to delete role_permissions: %v", err)
	}
	if err := db.Exec("DELETE FROM user_permissions").Error; err != nil {
		log.Fatalf("Failed to delete user_permissions: %v", err)
	}
	if err := db.Exec("DELETE FROM users").Error; err != nil {
		log.Fatalf("Failed to delete users: %v", err)
	}
	if err := db.Exec("DELETE FROM roles").Error; err != nil {
		log.Fatalf("Failed to delete roles: %v", err)
	}
	if err := db.Exec("DELETE FROM permissions").Error; err != nil {
		log.Fatalf("Failed to delete permissions: %v", err)
	}

	roles := []model.Role{
		{UUID: uuid.NewString(), Name: "admin"},
		{UUID: uuid.NewString(), Name: "user"},
	}

	crudUser := []model.Permission{
		{UUID: uuid.NewString(), Name: "read-user"},
		{UUID: uuid.NewString(), Name: "write-user"},
		{UUID: uuid.NewString(), Name: "delete-user"},
		{UUID: uuid.NewString(), Name: "update-user"},
	}

	crudPermissions := []model.Permission{
		{UUID: uuid.NewString(), Name: "read-permission"},
		{UUID: uuid.NewString(), Name: "write-permission"},
		{UUID: uuid.NewString(), Name: "delete-permission"},
		{UUID: uuid.NewString(), Name: "update-permission"},
	}

	crudRole := []model.Permission{
		{UUID: uuid.NewString(), Name: "read-role"},
		{UUID: uuid.NewString(), Name: "write-role"},
		{UUID: uuid.NewString(), Name: "delete-role"},
		{UUID: uuid.NewString(), Name: "update-role"},
	}

	otherPermission := model.Permission{
		UUID: uuid.NewString(), Name: "read-other",
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

	users := []model.User{
		{UUID: uuid.NewString(), Name: "Test", Email: "test@test.com", Password: string(hashedPassword)},
	}

	// Seed roles
	for _, role := range roles {
		if err := db.Create(&role).Error; err != nil {
			log.Fatalf("Failed to seed roles: %v", err)
		}
	}

	// Seed user permissions
	for _, permission := range crudUser {
		if err := db.Create(&permission).Error; err != nil {
			log.Fatalf("Failed to seed permissions: %v", err)
		}
	}

	// Seed crud permissions
	for _, permission := range crudPermissions {
		if err := db.Create(&permission).Error; err != nil {
			log.Fatalf("Failed to seed permissions: %v", err)
		}
	}

	// Seed role permissions
	for _, permission := range crudRole {
		if err := db.Create(&permission).Error; err != nil {
			log.Fatalf("Failed to seed permissions: %v", err)
		}
	}

	if err := db.Create(&otherPermission).Error; err != nil {
		log.Fatalf("Failed to seed permissions: %v", err)
	}

	// Seed users
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			log.Fatalf("Failed to seed users: %v", err)
		}
	}

	// Assign permissions to roles
	adminRole := roles[0]
	if err := db.Model(&adminRole).Association("Permissions").Append(&crudPermissions); err != nil {
		log.Fatalf("Failed to assign permissions to admin role: %v", err)
	}

	if err := db.Model(&adminRole).Association("Permissions").Append(&crudRole); err != nil {
		log.Fatalf("Failed to assign permissions to admin role: %v", err)
	}

	userRole := roles[1]
	if err := db.Model(&userRole).Association("Permissions").Append(&crudUser); err != nil {
		log.Fatalf("Failed to assign permissions to admin role: %v", err)
	}

	// Assign roles to users
	testUser := users[0]
	if err := db.Model(&testUser).Association("Roles").Append(&adminRole); err != nil {
		log.Fatalf("Failed to assign admin role to test user: %v", err)
	}

	if err := db.Model(&testUser).Association("Roles").Append(&userRole); err != nil {
		log.Fatalf("Failed to assign admin role to test user: %v", err)
	}

	if err := db.Model(&testUser).Association("Permissions").Append([]model.Permission{
		otherPermission,
	}); err != nil {
		log.Fatalf("Failed to assign update permission to test user: %v", err)
	}
}
