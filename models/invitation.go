package models

import "time"

// Invitation represents an invite allowing magic-link authentication
// Status: pending | sent | used | revoked | expired
// Token is a random secret used in magic links
// ExpiresAt defaults to 7 days after creation
// UsedAt is set when first redeemed
// Index email for quick lookups and enforce single active pending per email in app logic
// We do not store password, only one-time tokens and emails
// Token should be unique
// Note: We avoid soft-deletes to maintain audit history
// Minimal model to stay simple and secure
//
// Application must ensure token is single-use and invalidated upon login
// via setting Status to "used" and UsedAt
// and by disallowing reuse in handlers

type Invitation struct {
	ID        uint      `gorm:"primarykey"`
	Email     string    `gorm:"index;not null"`
	Token     string    `gorm:"uniqueIndex;not null"`
	Status    string    `gorm:"not null;default:pending"`
	CreatedAt time.Time
	ExpiresAt time.Time `gorm:"index"`
	UsedAt    *time.Time
}