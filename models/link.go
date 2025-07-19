package models

import (
	"time"

	"gorm.io/gorm"
)

type Link struct {
	ID          uint           `gorm:"primarykey"`
	Alias       string         `gorm:"uniqueIndex:idx_alias_deleted;not null"`
	URL         string         `gorm:"not null"`
	Clicks      uint           `gorm:"default:0"`
	CreatorName string         `gorm:"not null"`
	CreatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"uniqueIndex:idx_alias_deleted"`
}