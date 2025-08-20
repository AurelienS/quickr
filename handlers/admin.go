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

// InviteRow is a view model for rendering an invitation row with user-level disabled flag
type InviteRow struct {
	models.Invitation
	UserDisabled bool
}

// buildInviteRows annotates invitations with user disabled status
func buildInviteRows(db *gorm.DB, invites []models.Invitation) []InviteRow {
	rows := make([]InviteRow, 0, len(invites))
	// Build a set of emails to query once
	emailSet := map[string]struct{}{}
	for _, inv := range invites {
		emailSet[inv.Email] = struct{}{}
	}
	// Fetch users for these emails
	var users []models.User
	if len(emailSet) > 0 {
		emails := make([]string, 0, len(emailSet))
		for e := range emailSet {
			emails = append(emails, e)
		}
		db.Where("email IN ?", emails).Find(&users)
	}
	// Map email -> disabled
	emailToDisabled := map[string]bool{}
	for _, u := range users {
		emailToDisabled[u.Email] = u.Disabled
	}
	for _, inv := range invites {
		rows = append(rows, InviteRow{Invitation: inv, UserDisabled: emailToDisabled[inv.Email]})
	}
	return rows
}

// GET /admin renders a simple dashboard (list + create form)
func AdminDashboard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var invites []models.Invitation
		db.Order("created_at desc").Limit(200).Find(&invites)
		emailVal, _ := c.Get("userEmail")
		roleVal, _ := c.Get("userRole")
		isAdmin := roleVal == "admin"
		c.HTML(http.StatusOK, "admin.html", gin.H{
			"active":    "admin",
			"invites":   buildInviteRows(db, invites),
			"userEmail": emailVal,
			"isAdmin":   isAdmin,
		})
	}
}

// POST /admin/invitations creates an invitation and sends it immediately
func CreateInvitation(db *gorm.DB, mailer *SendinblueClient, appBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}
		log.Printf("[ADMIN] CreateInvitation for %s", email)
		// Revoke any active invites
		db.Model(&models.Invitation{}).Where("email = ? AND status IN ?", email, []string{"pending", "sent"}).Update("status", "revoked")
		// Re-enable user if previously disabled
		var user models.User
		if err := db.Where("email = ?", email).First(&user).Error; err == nil {
			if user.Disabled {
				user.Disabled = false
				if err := db.Save(&user).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to re-enable user"})
					return
				}
			}
		}
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

		// Auto-send the invitation immediately
		if inv.Status != "used" && inv.Status != "revoked" {
			base := resolveBaseURL(c.Request, appBaseURL)
			link := base + "/magic?token=" + inv.Token
			log.Printf("[ADMIN] Auto-sending magic link to %s", inv.Email)
			if err := mailer.SendMagicLink(inv.Email, link); err != nil {
				log.Printf("[ADMIN] Auto-send failed for %s: %v", inv.Email, err)
				// keep status as pending so admin can retry via Send
			} else {
				inv.Status = "sent"
				if err := db.Save(&inv).Error; err != nil {
					log.Printf("[ADMIN] Failed to mark invite sent id=%d: %v", inv.ID, err)
				}
			}
		}

		if c.GetHeader("HX-Request") == "true" {
			var row InviteRow
			{
				var u models.User
				if err := db.Where("email = ?", inv.Email).First(&u).Error; err == nil {
					row = InviteRow{Invitation: inv, UserDisabled: u.Disabled}
				} else {
					row = InviteRow{Invitation: inv, UserDisabled: false}
				}
			}
			c.HTML(http.StatusCreated, "admin_invite_row.html", row)
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
		if inv.Status == "used" || inv.Status == "revoked" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot revoke this invite"})
			return
		}
		inv.Status = "revoked"
		if err := db.Save(&inv).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke"})
			return
		}
		if c.GetHeader("HX-Request") == "true" {
			var row InviteRow
			{
				var u models.User
				if err := db.Where("email = ?", inv.Email).First(&u).Error; err == nil {
					row = InviteRow{Invitation: inv, UserDisabled: u.Disabled}
				} else {
					row = InviteRow{Invitation: inv, UserDisabled: false}
				}
			}
			c.HTML(http.StatusOK, "admin_invite_row.html", row)
			return
		}
		c.JSON(http.StatusOK, inv)
	}
}

// POST /admin/invitations/revoke-email disables the user and revokes all invites for that email
func RevokeInvitationsByEmail(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}
		log.Printf("[ADMIN] RevokeInvitationsByEmail email=%s", email)
		// Disable user account (create if not exists)
		var user models.User
		res := db.Where("email = ?", email).Limit(1).Find(&user)
		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lookup user"})
			return
		}
		if res.RowsAffected == 0 {
			// No user exists; create a disabled user record so UI correctly reflects revoked state
			user = models.User{Email: email, Role: "user", Disabled: true}
			if err := db.Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create disabled user"})
				return
			}
		} else {
			if user.Disabled {
				if c.GetHeader("HX-Request") == "true" {
					c.Header("HX-Reswap", "none")
					c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div id="invite-error" hx-swap-oob="true">User already revoked</div>`))
					return
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": "user already revoked"})
				return
			}
			user.Disabled = true
			if err := db.Save(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable user"})
				return
			}
		}
		// Revoke all invitations for the email
		db.Model(&models.Invitation{}).Where("email = ? AND status <> ?", email, "revoked").Update("status", "revoked")
		// Return refreshed tbody with all invites
		if c.GetHeader("HX-Request") == "true" {
			var invites []models.Invitation
			db.Order("created_at desc").Limit(200).Find(&invites)
			c.HTML(http.StatusOK, "admin_invites_body.html", gin.H{"invites": buildInviteRows(db, invites)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"email": email, "revoked": true})
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
		base := resolveBaseURL(c.Request, appBaseURL)
		link := base + "/magic?token=" + inv.Token
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
			var row InviteRow
			{
				var u models.User
				if err := db.Where("email = ?", inv.Email).First(&u).Error; err == nil {
					row = InviteRow{Invitation: inv, UserDisabled: u.Disabled}
				} else {
					row = InviteRow{Invitation: inv, UserDisabled: false}
				}
			}
			c.HTML(http.StatusOK, "admin_invite_row.html", row)
			return
		}
		c.JSON(http.StatusOK, inv)
	}
}