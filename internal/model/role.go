package model

type Role struct {
    UUID        string       `json:"uuid"`
    Name        string       `json:"name"`
    Permissions []Permission `json:"permissions"`
}
