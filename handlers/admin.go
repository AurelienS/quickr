package handlers

import (
	"log"
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
		log.Printf("[ADMIN] CreateInvitation for %s", email)
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
		log.Printf("[ADMIN] Invitation %d created for %s", inv.ID, inv.Email)

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

// POST /admin/invitations/:id/send generates token if needed, sends email, then marks as sent
func SendInvitation(db *gorm.DB, mailer *SendinblueClient, appBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var inv models.Invitation
		if err := db.First(&inv, c.Param("id")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		log.Printf("[ADMIN] SendInvitation id=%d email=%s status=%s", inv.ID, inv.Email, inv.Status)
		if inv.Status == "used" || inv.Status == "revoked" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot send this invite"})
			return
		}
		// Ensure token exists
		if inv.Token == "" {
			token, err := generateToken(32)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
				return
			}
			inv.Token = token
			inv.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
			if err := db.Save(&inv).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save invite"})
				return
			}
		}
		link := appBaseURL + "/magic?token=" + inv.Token
		log.Printf("[ADMIN] Sending magic link to %s", inv.Email)
		if err := mailer.SendMagicLink(inv.Email, link); err != nil {
			log.Printf("[ADMIN] Send failed for %s: %v", inv.Email, err)
			if c.GetHeader("HX-Request") == "true" {
				// Prevent swapping the row and show an out-of-band error
				c.Header("HX-Reswap", "none")
				c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div id="invite-error" hx-swap-oob="true">Failed to send email. Check SENDINBLUE_API_KEY.</div>`))
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email"})
			return
		}
		// Mark as sent only after successful email
		inv.Status = "sent"
		if err := db.Save(&inv).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update invite"})
			return
		}
		log.Printf("[ADMIN] Invitation %d marked sent", inv.ID)
		if c.GetHeader("HX-Request") == "true" {
			c.HTML(http.StatusOK, "admin_invite_row.html", inv)
			return
		}
		c.JSON(http.StatusOK, inv)
	}
}