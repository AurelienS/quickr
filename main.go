package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"quickr/handlers"
	"quickr/models"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/js/*.js static/css/*.css
var staticFS embed.FS

// Ensure database directory exists
func ensureDBDir() error {
	dbDir := "data"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}
	return nil
}

func requireEnv(key string) {
	if os.Getenv(key) == "" {
		log.Printf("Config warning: %s is not set", key)
	}
}

func main() {
	// Load .env if present
	_ = godotenv.Load()

	// Create data directory if it doesn't exist
	if err := ensureDBDir(); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	// Validate environment configuration (warn-only)
	requireEnv("JWT_SECRET")
	requireEnv("ADMIN_EMAIL")
	requireEnv("APP_BASE_URL")
	requireEnv("SENDINBLUE_API_KEY")

	// Initialize SQLite database
	dbPath := filepath.Join("data", "quickr.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&models.Link{}, &models.User{}, &models.Invitation{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Setup Gin router
	r := gin.New() // Don't use Default() as it already includes Logger
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Add our custom request logger
	r.Use(func(c *gin.Context) {
		log.Printf("[REQUEST] %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
		log.Printf("[RESPONSE] %s %s -> %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	})

	// Load HTML templates from embedded FS
	templ := template.Must(template.New("").ParseFS(templateFS, "templates/*.html"))
	r.SetHTMLTemplate(templ)
	log.Println("Templates loaded from embedded FS")

	// Serve static files from embedded FS
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal("Failed to create sub filesystem for static files:", err)
	}
	r.StaticFS("/static", http.FS(staticSubFS))
	log.Println("Static files route added from embedded FS")

	// dependencies
	mailer := handlers.NewSendinblueClient()
	rateLimiter := handlers.NewIPLimiter(20) // 20 requests per minute per IP for login
	appBaseURL := os.Getenv("APP_BASE_URL")
	if appBaseURL == "" {
		appBaseURL = "http://localhost:8080"
	}

	// Public auth routes
	r.GET("/login", handlers.ShowLogin())
	r.POST("/login", handlers.RequestMagicLink(db, rateLimiter, mailer, appBaseURL))
	r.GET("/magic", handlers.RedeemMagicLink(db))
	r.POST("/logout", handlers.Logout())

	// Web routes (require auth)
	r.GET("/", handlers.RequireAuth(db), handlers.HandleHome(db))
	r.GET("/stats", handlers.RequireAuth(db), handlers.HandleStats(db))
	r.GET("/hot", handlers.RequireAuth(db), handlers.HandleHot(db))

	// Redirect route with debug handler (keep public)
	r.GET("/go/:alias", func(c *gin.Context) {
		log.Printf("[DEBUG] About to call redirect handler for alias: %s", c.Param("alias"))
		handlers.HandleRedirect(db)(c)
	})

	// Root-level alias redirect. Must come after fixed routes.
	r.GET("/:alias", func(c *gin.Context) {
		alias := c.Param("alias")
		if handlers.IsReservedAliasPublic(alias) {
			c.Status(http.StatusNotFound)
			return
		}
		handlers.HandleRedirect(db)(c)
	})

	// Admin routes
	admin := r.Group("/admin", handlers.RequireAuth(db), handlers.RequireAdmin())
	{
		admin.GET("", handlers.AdminDashboard(db))
		admin.POST("/invitations", handlers.CreateInvitation(db, mailer, appBaseURL))
		admin.POST("/invitations/:id/send", handlers.SendInvitation(db, mailer, appBaseURL))
		admin.POST("/invitations/:id/revoke", handlers.RevokeInvitation(db))
		admin.POST("/invitations/revoke-email", handlers.RevokeInvitationsByEmail(db))
	}

	// API routes (require auth)
	api := r.Group("/api", handlers.RequireAuth(db))
	{
		api.GET("/links", handlers.ListLinks(db))
		api.POST("/links", handlers.CreateLink(db))
		api.GET("/links/:id/edit", handlers.GetLinkEditField(db))
		api.PUT("/links/:id", handlers.UpdateLink(db))
		api.DELETE("/links/:id", handlers.DeleteLink(db))
		api.GET("/search", handlers.SearchLinks(db))
	}

	log.Printf("Server starting on http://localhost:8080")
	// Start server
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}