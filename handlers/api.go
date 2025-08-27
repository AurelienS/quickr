package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"quickr/services"
)

type CreateLinkRequest struct {
	Alias string `json:"alias" binding:"required"`
	URL   string `json:"url" binding:"required"`
}

type UpdateLinkRequest struct {
	URL string `json:"url" binding:"required"`
}

// GET /api/links
func (h *AppHandler) ListLinks() gin.HandlerFunc {
	return func(c *gin.Context) {
		links, err := h.LinkService.ListLinks()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch links"})
			return
		}
		c.JSON(http.StatusOK, links)
	}
}

// POST /api/links
func (h *AppHandler) CreateLink() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get current user email from context
		emailVal, _ := c.Get("userEmail")
		creatorEmail, _ := emailVal.(string)
		roleVal, _ := c.Get("userRole")
		role, _ := roleVal.(string)
		creatorDisplay := creatorEmail
		if role == "admin" {
			creatorDisplay = getAdminName()
		}

		// Check if it's an HTMX request
		if c.GetHeader("HX-Request") == "true" {
			// Parse form data
			alias := c.PostForm("alias")
			url := c.PostForm("url")

			if alias == "" || url == "" {
				c.String(http.StatusBadRequest, "All fields are required")
				return
			}

			link, err := h.LinkService.CreateLink(alias, url, creatorDisplay)
			if err != nil {
				switch {
				case errors.Is(err, services.ErrAliasReserved):
					c.String(http.StatusBadRequest, "Alias is reserved")
				case errors.Is(err, services.ErrInvalidURL):
					c.String(http.StatusBadRequest, "Invalid URL format")
				case errors.Is(err, services.ErrAliasExists):
					c.String(http.StatusConflict, "Alias already exists")
				default:
					c.String(http.StatusInternalServerError, "Failed to create link")
				}
				return
			}
			// Return just the new row HTML
			c.HTML(http.StatusCreated, "link_row.html", *link)
			return
		}

		// Handle regular JSON API request
		var req CreateLinkRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		link, err := h.LinkService.CreateLink(req.Alias, req.URL, creatorDisplay)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrAliasReserved):
				c.JSON(http.StatusBadRequest, gin.H{"error": "Alias is reserved"})
			case errors.Is(err, services.ErrInvalidURL):
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL format"})
			case errors.Is(err, services.ErrAliasExists):
				c.JSON(http.StatusConflict, gin.H{"error": "Alias already exists"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create link"})
			}
			return
		}

		c.JSON(http.StatusCreated, link)
	}
}

// GET /api/links/modal/create
func (h *AppHandler) GetCreateLinkModal() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "modal_create_link.html", nil)
	}
}

// GET /api/links/:id/modal/edit
func (h *AppHandler) GetLinkEditModal() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		link, err := h.LinkService.GetLinkByID(id)
		if err != nil {
			c.String(http.StatusNotFound, "Link not found")
			return
		}
		c.HTML(http.StatusOK, "modal_edit_link.html", link)
	}
}

// GET /api/links/:id/modal/delete
func (h *AppHandler) GetLinkDeleteModal() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		link, err := h.LinkService.GetLinkByID(id)
		if err != nil {
			c.String(http.StatusNotFound, "Link not found")
			return
		}
		c.HTML(http.StatusOK, "modal_delete_link.html", link)
	}
}

// GET /api/links/:id/edit
func (h *AppHandler) GetLinkEditField() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		field := c.Query("field")

		if field != "alias" && field != "url" {
			c.String(http.StatusBadRequest, "Invalid field")
			return
		}

		link, err := h.LinkService.GetLinkByID(id)
		if err != nil {
			c.String(http.StatusNotFound, "Link not found")
			return
		}

		var value string
		if field == "alias" {
			value = link.Alias
		} else {
			value = link.URL
		}

		c.HTML(http.StatusOK, "link_edit.html", gin.H{
			"id":    link.ID,
			"field": field,
			"value": value,
		})
	}
}

// PUT /api/links/:id
func (h *AppHandler) UpdateLink() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Determine editor
		emailVal, _ := c.Get("userEmail")
		editorEmail, _ := emailVal.(string)
		roleVal, _ := c.Get("userRole")
		role, _ := roleVal.(string)
		editorDisplay := editorEmail
		if role == "admin" {
			editorDisplay = getAdminName()
		}

		newAlias := c.PostForm("alias")
		newURL := c.PostForm("url")
		updated, err := h.LinkService.UpdateLink(id, newAlias, newURL, editorDisplay)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrAliasReserved):
				c.String(http.StatusBadRequest, "Alias is reserved")
			case errors.Is(err, services.ErrInvalidURL):
				c.String(http.StatusBadRequest, "Invalid URL format")
			case errors.Is(err, services.ErrAliasExists):
				c.String(http.StatusConflict, "Alias already exists")
			case errors.Is(err, services.ErrLinkNotFound):
				c.String(http.StatusNotFound, "Link not found")
			default:
				c.String(http.StatusInternalServerError, "Failed to update link")
			}
			return
		}

		// Return updated HTML for HTMX
		if c.GetHeader("HX-Request") == "true" {
			if newAlias != "" && newURL != "" {
				c.HTML(http.StatusOK, "link_row.html", updated)
			} else if newAlias != "" {
				c.HTML(http.StatusOK, "link_cell.html", gin.H{
					"id":    updated.ID,
					"field": "alias",
					"value": updated.Alias,
				})
			} else if newURL != "" {
				c.HTML(http.StatusOK, "link_cell.html", gin.H{
					"id":    updated.ID,
					"field": "url",
					"value": updated.URL,
					"alias": updated.Alias,
				})
			} else {
				c.HTML(http.StatusOK, "link_row.html", updated)
			}
			return
		}

		c.JSON(http.StatusOK, updated)
	}
}

// DELETE /api/links/:id
func (h *AppHandler) DeleteLink() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := h.LinkService.DeleteLink(c.Param("id"))
		if err != nil {
			if errors.Is(err, services.ErrLinkNotFound) {
				c.String(http.StatusNotFound, "Link not found")
			} else {
				c.String(http.StatusInternalServerError, "Failed to delete link")
			}
			return
		}

		// If it's an HTMX request, return the updated list
		if c.GetHeader("HX-Request") == "true" {
			links, _ := h.LinkService.ListLinks()
			c.HTML(http.StatusOK, "link_rows.html", gin.H{
				"links": links,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Link deleted successfully"})
	}
}

// GET /api/search
func (h *AppHandler) SearchLinks() gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			ls, e := h.LinkService.ListLinks()
			if e != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
			for i := range ls {
				c.HTML(http.StatusOK, "link_row.html", ls[i])
			}
			return
		}
		ls, e := h.LinkService.SearchLinks(query)
		if e != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		for i := range ls {
			c.HTML(http.StatusOK, "link_row.html", ls[i])
		}
	}
}