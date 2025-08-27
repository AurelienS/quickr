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
	"quickr/domain/reserved"
	"quickr/handlers"
	infraMailer "quickr/infrastructure/mailer"
	"quickr/infrastructure/ratelimit"
	"quickr/interfaces/session"
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
	_ = godotenv.Load()
	must(ensureDBDir())
	warnEnv()

	db := mustDB()
	mustMigrate(db)

	r := newRouter()
	loadTemplates(r)
	mountStatic(r)

	h := wireHandlers(db)
	registerRoutes(r, h)

	start(r)
}

func warnEnv() {
	requireEnv("JWT_SECRET")
	requireEnv("ADMIN_EMAIL")
	requireEnv("APP_BASE_URL")
	requireEnv("SENDINBLUE_API_KEY")
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func mustDB() *gorm.DB {
	dbPath := filepath.Join("data", "quickr.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	return db
}

func mustMigrate(db *gorm.DB) {
	if err := db.AutoMigrate(&models.Link{}, &models.User{}, &models.Invitation{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}

func newRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(func(c *gin.Context) {
		log.Printf("[REQUEST] %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
		log.Printf("[RESPONSE] %s %s -> %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	})
	return r
}

func loadTemplates(r *gin.Engine) {
	templ := template.Must(template.New("").ParseFS(templateFS, "templates/*.html"))
	r.SetHTMLTemplate(templ)
	log.Println("Templates loaded from embedded FS")
}

func mountStatic(r *gin.Engine) {
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal("Failed to create sub filesystem for static files:", err)
	}
	r.StaticFS("/static", http.FS(staticSubFS))
	log.Println("Static files route added from embedded FS")
}

func wireHandlers(db *gorm.DB) *handlers.AppHandler {
	emailSender := infraMailer.NewSendinblueClient()
	rateLimiter := ratelimit.NewIPLimiter(20) // 20 requests per minute per IP for login
	appBaseURL := getenvDefault("APP_BASE_URL", "http://localhost:8080")

	linkRepo := repositories.NewGormLinkRepository(db)
	userRepo := repositories.NewGormUserRepository(db)
	invRepo := repositories.NewGormInvitationRepository(db)
	linkService := services.NewLinkService(linkRepo)
	authService := services.NewAuthService(userRepo, invRepo, emailSender, appBaseURL, nil)
	statsService := services.NewStatsService(linkService)
	jwtSecret := os.Getenv("JWT_SECRET")
	sess := session.NewManager([]byte(jwtSecret), "session", 180*24*60*60*1e9)
	return handlers.NewAppHandler(linkService, authService, statsService, rateLimiter, appBaseURL, sess)
}

func registerRoutes(r *gin.Engine, h *handlers.AppHandler) {
	// Public auth routes
	r.GET("/login", h.ShowLogin())
	r.POST("/login", h.RequestMagicLink())
	r.GET("/magic", h.RedeemMagicLink())
	r.POST("/logout", handlers.Logout())

	// Web routes (require auth)
	r.GET("/", h.RequireAuth(), h.HandleHome())
	r.GET("/stats", h.RequireAuth(), h.HandleStats())
	r.GET("/hot", h.RequireAuth(), h.HandleHot())

	// Redirect route with debug handler (keep public)
	r.GET("/go/:alias", func(c *gin.Context) {
		log.Printf("[DEBUG] About to call redirect handler for alias: %s", c.Param("alias"))
		h.HandleRedirect()(c)
	})

	// Root-level alias redirect. Must come after fixed routes.
	r.GET("/:alias", func(c *gin.Context) {
		alias := c.Param("alias")
		if reserved.IsReservedAlias(alias) {
			c.Status(http.StatusNotFound)
			return
		}
		h.HandleRedirect()(c)
	})

	// Admin routes
	admin := r.Group("/admin", h.RequireAuth(), h.RequireAdmin())
	{
		admin.GET("", h.AdminDashboard())
		admin.POST("/invitations", h.CreateInvitation())
		admin.POST("/invitations/:id/send", h.SendInvitation())
		admin.POST("/invitations/:id/revoke", h.RevokeInvitation())
		admin.POST("/invitations/revoke-email", h.RevokeInvitationsByEmail())
	}

	// API routes (require auth)
	api := r.Group("/api", h.RequireAuth())
	{
		api.GET("/links", h.ListLinks())
		api.POST("/links", h.CreateLink())
		api.GET("/links/modal/create", h.GetCreateLinkModal())
		api.GET("/links/:id/modal/edit", h.GetLinkEditModal())
		api.GET("/links/:id/modal/delete", h.GetLinkDeleteModal())
		api.GET("/links/:id/edit", h.GetLinkEditField())
		api.PUT("/links/:id", h.UpdateLink())
		api.DELETE("/links/:id", h.DeleteLink())
		api.GET("/search", h.SearchLinks())
	}
}

func start(r *gin.Engine) {
	log.Printf("Server starting on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}