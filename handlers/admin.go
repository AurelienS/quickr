package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"quickr/models"
)

// InviteRow is a view model for rendering an invitation row with user-level disabled flag
type InviteRow struct {
	models.Invitation
	UserDisabled bool
}

// GET /admin renders a simple dashboard (list + create form)
func (h *AppHandler) AdminDashboard() gin.HandlerFunc {
	return func(c *gin.Context) {
		invites, _ := h.AuthService.ListInvitations(200)
		annotated, _ := h.AuthService.AnnotateInvitesWithUserDisabled(invites)
		rows := make([]InviteRow, 0, len(annotated))
		for _, a := range annotated { rows = append(rows, InviteRow{Invitation: a.Inv, UserDisabled: a.UserDisabled}) }
		emailVal, _ := c.Get("userEmail")
		roleVal, _ := c.Get("userRole")
		isAdmin := roleVal == "admin"
		c.HTML(http.StatusOK, "admin.html", gin.H{
			"active":    "admin",
			"invites":   rows,
			"userEmail": emailVal,
			"isAdmin":   isAdmin,
		})
	}
}

// POST /admin/invitations creates an invitation and sends it immediately
func (h *AppHandler) CreateInvitation() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}
		log.Printf("[ADMIN] CreateInvitation for %s", email)
		base := resolveBaseURL(c.Request, h.AppBaseURL)
		if _, err := h.AuthService.CreateMagicLinkInvite(email, base); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invitation"})
			return
		}
		// Render a fresh row
		invites, _ := h.AuthService.ListInvitations(1)
		var inv models.Invitation
		if len(invites) > 0 { inv = invites[0] }
		if c.GetHeader("HX-Request") == "true" {
			annotated, _ := h.AuthService.AnnotateInvitesWithUserDisabled([]models.Invitation{inv})
			row := InviteRow{Invitation: inv, UserDisabled: len(annotated) == 1 && annotated[0].UserDisabled}
			c.HTML(http.StatusCreated, "admin_invite_row.html", row)
			return
		}
		c.JSON(http.StatusCreated, inv)
	}
}

// POST /admin/invitations/:id/revoke toggles to revoked and returns row
func (h *AppHandler) RevokeInvitation() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		inv, err := h.AuthService.RevokeInvitationByID(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if c.GetHeader("HX-Request") == "true" {
			annotated, _ := h.AuthService.AnnotateInvitesWithUserDisabled([]models.Invitation{*inv})
			row := InviteRow{Invitation: *inv, UserDisabled: len(annotated) == 1 && annotated[0].UserDisabled}
			c.HTML(http.StatusOK, "admin_invite_row.html", row)
			return
		}
		c.JSON(http.StatusOK, inv)
	}
}

// POST /admin/invitations/revoke-email disables the user and revokes all invites for that email
func (h *AppHandler) RevokeInvitationsByEmail() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}
		log.Printf("[ADMIN] RevokeInvitationsByEmail email=%s", email)
		if err := h.AuthService.DisableUser(email); err != nil {
			if err.Error() == "user already revoked" {
				if c.GetHeader("HX-Request") == "true" {
					c.Header("HX-Reswap", "none")
					c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div id="invite-error" hx-swap-oob="true">User already revoked</div>`))
					return
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": "user already revoked"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable user"})
			return
		}
		_ = h.AuthService.RevokeAllForEmail(email)
		if c.GetHeader("HX-Request") == "true" {
			invites, _ := h.AuthService.ListInvitations(200)
			annotated, _ := h.AuthService.AnnotateInvitesWithUserDisabled(invites)
			rows := make([]InviteRow, 0, len(annotated))
			for _, a := range annotated { rows = append(rows, InviteRow{Invitation: a.Inv, UserDisabled: a.UserDisabled}) }
			c.HTML(http.StatusOK, "admin_invites_body.html", gin.H{"invites": rows})
			return
		}
		c.JSON(http.StatusOK, gin.H{"email": email, "revoked": true})
	}
}

// POST /admin/invitations/:id/send generates token if needed, sends email, then marks as sent
func (h *AppHandler) SendInvitation() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		base := resolveBaseURL(c.Request, h.AppBaseURL)
		inv, err := h.AuthService.SendInvitationByID(id, base)
		if err != nil {
			log.Printf("[ADMIN] Send failed for id=%s: %v", id, err)
			if c.GetHeader("HX-Request") == "true" {
				c.Header("HX-Reswap", "none")
				c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div id="invite-error" hx-swap-oob="true">Failed to send email. Check SENDINBLUE_API_KEY.</div>`))
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email"})
			return
		}
		if c.GetHeader("HX-Request") == "true" {
			annotated, _ := h.AuthService.AnnotateInvitesWithUserDisabled([]models.Invitation{*inv})
			row := InviteRow{Invitation: *inv, UserDisabled: len(annotated) == 1 && annotated[0].UserDisabled}
			c.HTML(http.StatusOK, "admin_invite_row.html", row)
			return
		}
		c.JSON(http.StatusOK, inv)
	}
}