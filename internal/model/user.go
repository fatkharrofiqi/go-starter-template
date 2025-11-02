package model

import (
    "time"
)

type User struct {
    UUID        string       `json:"uuid"`
    Email       string       `json:"email"`
    Password    string       `json:"-,omitempty"`
    Name        string       `json:"name"`
    Roles       []Role       `json:"roles"`
    Permissions []Permission `json:"permissions"`
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
    DeletedAt   *time.Time   `json:"deleted_at"`
}
