package model

type Role struct {
	UUID        string       `gorm:"primaryKey;unique;not null" json:"uuid"`
	Name        string       `gorm:"unique;not null" json:"name"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions"`
}
