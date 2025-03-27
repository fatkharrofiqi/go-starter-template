package model

type Permission struct {
	UUID string `gorm:"primaryKey;unique;not null" json:"uuid"`
	Name string `gorm:"unique;not null" json:"name"`
}
