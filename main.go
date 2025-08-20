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
	"quickr/repositories"
	"quickr/services"
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

	// repositories and services (DI)
	linkRepo := repositories.NewGormLinkRepository(db)
	linkService := services.NewLinkService(linkRepo)
	userRepo := repositories.NewGormUserRepository(db)
	invRepo := repositories.NewGormInvitationRepository(db)
	authService := services.NewAuthService(userRepo, invRepo, mailer, appBaseURL, nil)
	h := handlers.NewAppHandler(linkService, authService, mailer, rateLimiter, appBaseURL)

	// Public auth routes
	r.GET("/login", h.ShowLogin())
	r.POST("/login", h.RequestMagicLink(db))
	r.GET("/magic", h.RedeemMagicLink(db))
	r.POST("/logout", handlers.Logout())

	// Web routes (require auth)
	r.GET("/", h.RequireAuth(db), h.HandleHome())
	r.GET("/stats", h.RequireAuth(db), h.HandleStats())
	r.GET("/hot", h.RequireAuth(db), h.HandleHot())

	// Redirect route with debug handler (keep public)
	r.GET("/go/:alias", func(c *gin.Context) {
		log.Printf("[DEBUG] About to call redirect handler for alias: %s", c.Param("alias"))
		h.HandleRedirect(db)(c)
	})

	// Root-level alias redirect. Must come after fixed routes.
	r.GET("/:alias", func(c *gin.Context) {
		alias := c.Param("alias")
		if handlers.IsReservedAliasPublic(alias) {
			c.Status(http.StatusNotFound)
			return
		}
		h.HandleRedirect(db)(c)
	})

	// Admin routes
	admin := r.Group("/admin", h.RequireAuth(db), h.RequireAdmin())
	{
		admin.GET("", h.AdminDashboard(db))
		admin.POST("/invitations", h.CreateInvitation(db))
		admin.POST("/invitations/:id/send", h.SendInvitation(db))
		admin.POST("/invitations/:id/revoke", h.RevokeInvitation(db))
		admin.POST("/invitations/revoke-email", h.RevokeInvitationsByEmail(db))
	}

	// API routes (require auth)
	api := r.Group("/api", h.RequireAuth(db))
	{
		api.GET("/links", h.ListLinks())
		api.POST("/links", h.CreateLink())
		api.GET("/links/:id/edit", h.GetLinkEditField())
		api.PUT("/links/:id", h.UpdateLink())
		api.DELETE("/links/:id", h.DeleteLink())
		api.GET("/search", h.SearchLinks())
	}

	log.Printf("Server starting on http://localhost:8080")
	// Start server
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}