package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	UUID        string         `gorm:"primaryKey;unique;not null" json:"uuid"`
	Email       string         `gorm:"unique" json:"email"`
	Password    string         `json:"-,omitempty"`
	Name        string         `json:"name"`
	Roles       []Role         `gorm:"many2many:user_roles;" json:"roles"`
	Permissions []Permission   `gorm:"many2many:user_permissions;" json:"permissions"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
