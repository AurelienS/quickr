package models

import "time"

// User represents an authenticated user of the system
// Role is either "admin" or "user"
type User struct {
	ID        uint      `gorm:"primarykey"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Role      string    `gorm:"not null;default:user"`
	CreatedAt time.Time
	LastLogin time.Time
}