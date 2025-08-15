package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quickr/models"
)

// GET /admin renders a simple dashboard (list + create form)
func AdminDashboard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var invites []models.Invitation
		db.Order("created_at desc").Limit(200).Find(&invites)
		c.HTML(http.StatusOK, "admin.html", gin.H{
			"invites": invites,
		})
	}
}

// POST /admin/invitations creates an invitation
func CreateInvitation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}
		// Revoke any active invites
		db.Model(&models.Invitation{}).Where("email = ? AND status IN ?", email, []string{"pending", "sent"}).Update("status", "revoked")
		// Create pending invite with a unique placeholder token to satisfy UNIQUE constraint
		token, err := generateToken(32)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}
		inv := models.Invitation{
			Email:     email,
			Token:     token,
			Status:    "pending",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		if err := db.Create(&inv).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invitation"})
			return
		}

		if c.GetHeader("HX-Request") == "true" {
			c.HTML(http.StatusCreated, "admin_invite_row.html", inv)
			return
		}
		c.JSON(http.StatusCreated, inv)
	}
}

// POST /admin/invitations/:id/revoke toggles to revoked and returns row
func RevokeInvitation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var inv models.Invitation
		if err := db.First(&inv, c.Param("id")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		inv.Status = "revoked"
		if err := db.Save(&inv).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke"})
			return
		}
		if c.GetHeader("HX-Request") == "true" {
			c.HTML(http.StatusOK, "admin_invite_row.html", inv)
			return
		}
		c.JSON(http.StatusOK, inv)
	}
}

// POST /admin/invitations/:id/send generates token, marks as sent, returns row
func SendInvitation(db *gorm.DB, mailer *SendinblueClient, appBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var inv models.Invitation
		if err := db.First(&inv, c.Param("id")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if inv.Status == "used" || inv.Status == "revoked" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot send this invite"})
			return
		}
		token, err := generateToken(32)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}
		inv.Token = token
		inv.Status = "sent"
		inv.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
		if err := db.Save(&inv).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save invite"})
			return
		}
		link := appBaseURL + "/magic?token=" + token
		_ = mailer.SendMagicLink(inv.Email, link)
		if c.GetHeader("HX-Request") == "true" {
			c.HTML(http.StatusOK, "admin_invite_row.html", inv)
			return
		}
		c.JSON(http.StatusOK, inv)
	}
}