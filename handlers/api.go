package handlers

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quickr/models"
)

type CreateLinkRequest struct {
	Alias string `json:"alias" binding:"required"`
	URL   string `json:"url" binding:"required"`
}

type UpdateLinkRequest struct {
	URL string `json:"url" binding:"required"`
}

// Validation helper
func validateURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}

// GET /api/links
func ListLinks(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var links []models.Link
		result := db.Find(&links)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch links"})
			return
		}
		c.JSON(http.StatusOK, links)
	}
}

// POST /api/links
func CreateLink(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get current user email from context
		emailVal, _ := c.Get("userEmail")
		creatorEmail, _ := emailVal.(string)

		// Check if it's an HTMX request
		if c.GetHeader("HX-Request") == "true" {
			// Parse form data
			alias := c.PostForm("alias")
			url := c.PostForm("url")

			if alias == "" || url == "" {
				c.String(http.StatusBadRequest, "All fields are required")
				return
			}

			// Validate URL
			if !validateURL(url) {
				c.String(http.StatusBadRequest, "Invalid URL format")
				return
			}

			// Prevent reserved routes
			if isReservedAlias(alias) {
				c.String(http.StatusBadRequest, "Alias is reserved")
				return
			}
			// Check for duplicate alias (only among non-deleted links)
			var existing models.Link
			if err := db.Where("alias = ?", alias).First(&existing).Error; err == nil {
				c.String(http.StatusConflict, "Alias already exists")
				return
			}

			link := models.Link{
				Alias:       alias,
				URL:         url,
				CreatorName: creatorEmail,
			}

			if err := db.Create(&link).Error; err != nil {
				c.String(http.StatusInternalServerError, "Failed to create link")
				return
			}

			// Return just the new row HTML
			c.HTML(http.StatusCreated, "link_row.html", link)
			return
		}

		// Handle regular JSON API request
		var req CreateLinkRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Validate URL
		if !validateURL(req.URL) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL format"})
			return
		}

		// Prevent reserved routes
		if isReservedAlias(req.Alias) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Alias is reserved"})
			return
		}
		// Check for duplicate alias (only among non-deleted links)
		var existing models.Link
		if err := db.Where("alias = ?", req.Alias).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Alias already exists"})
			return
		}

		link := models.Link{
			Alias:       req.Alias,
			URL:         req.URL,
			CreatorName: creatorEmail,
		}

		if err := db.Create(&link).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create link"})
			return
		}

		c.JSON(http.StatusCreated, link)
	}
}

// GET /api/links/:id/edit
func GetLinkEditField(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		field := c.Query("field")

		if field != "alias" && field != "url" {
			c.String(http.StatusBadRequest, "Invalid field")
			return
		}

		var link models.Link
		if err := db.First(&link, id).Error; err != nil {
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
func UpdateLink(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Determine editor
		emailVal, _ := c.Get("userEmail")
		editorEmail, _ := emailVal.(string)

		var link models.Link
		if err := db.First(&link, id).Error; err != nil {
			c.String(http.StatusNotFound, "Link not found")
			return
		}

		// Check which field is being updated
		if alias := c.PostForm("alias"); alias != "" {
			// Prevent reserved routes
			if isReservedAlias(alias) {
				c.String(http.StatusBadRequest, "Alias is reserved")
				return
			}
			// Check for duplicate alias (only among non-deleted links)
			var existing models.Link
			if err := db.Where("alias = ? AND id != ?", alias, id).First(&existing).Error; err == nil {
				c.String(http.StatusConflict, "Alias already exists")
				return
			}
			link.Alias = alias
		}

		if url := c.PostForm("url"); url != "" {
			if !validateURL(url) {
				c.String(http.StatusBadRequest, "Invalid URL format")
				return
			}
			link.URL = url
		}

		// Update last editor as creatorName
		if editorEmail != "" {
			link.CreatorName = editorEmail
		}

		if err := db.Save(&link).Error; err != nil {
			c.String(http.StatusInternalServerError, "Failed to update link")
			return
		}

		// Return the updated cell HTML
		if c.GetHeader("HX-Request") == "true" {
			if alias := c.PostForm("alias"); alias != "" {
				c.HTML(http.StatusOK, "link_cell.html", gin.H{
					"id":    link.ID,
					"field": "alias",
					"value": link.Alias,
				})
			} else if url := c.PostForm("url"); url != "" {
				c.HTML(http.StatusOK, "link_cell.html", gin.H{
					"id":    link.ID,
					"field": "url",
					"value": link.URL,
					"alias": link.Alias,
				})
			}
			return
		}

		c.JSON(http.StatusOK, link)
	}
}

// DELETE /api/links/:id
func DeleteLink(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var link models.Link
		if err := db.First(&link, c.Param("id")).Error; err != nil {
			c.String(http.StatusNotFound, "Link not found")
			return
		}

		if err := db.Delete(&link).Error; err != nil {
			c.String(http.StatusInternalServerError, "Failed to delete link")
			return
		}

		// If it's an HTMX request, return the updated list
		if c.GetHeader("HX-Request") == "true" {
			var links []models.Link
			db.Order("created_at desc").Find(&links)
			c.HTML(http.StatusOK, "link_rows.html", gin.H{
				"links": links,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Link deleted successfully"})
	}
}

// GET /api/search
func SearchLinks(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        query := c.Query("q")
        if query == "" {
            // If no query, return all links
            var links []models.Link
            db.Order("created_at desc").Find(&links)
            // Use the same template as the main list
            for i := range links {
                c.HTML(http.StatusOK, "link_row.html", links[i])
            }
            return
        }

        // Search only in alias and url
        var links []models.Link
        searchQuery := "%" + query + "%"
        db.Where("alias LIKE ? OR url LIKE ?", searchQuery, searchQuery).
            Order("created_at desc").
            Find(&links)

        // Use the same template as the main list
        for i := range links {
            c.HTML(http.StatusOK, "link_row.html", links[i])
        }
    }
}