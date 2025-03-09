package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	UUID     string `gorm:"unique;not null" json:"uuid"`
	Email    string `gorm:"unique" json:"email"`
	Password string `json:"-,omitempty"`
	Name     string `json:"name"`
}
