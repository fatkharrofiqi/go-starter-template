package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	UID      string `gorm:"unique;not null" json:"uid"`
	Email    string `gorm:"unique" json:"email"`
	Password string `json:"-,omitempty"`
	Name     string `json:"name"`
}
