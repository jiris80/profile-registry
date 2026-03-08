package model

import "time"

type Record struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"-"`
	ExternalID  string    `gorm:"uniqueIndex;not null"     json:"external_id"`
	Name        string    `gorm:"not null"                 json:"name"`
	Email       string    `gorm:"not null"                 json:"email"`
	DateOfBirth time.Time `gorm:"not null"                 json:"date_of_birth"`
}
