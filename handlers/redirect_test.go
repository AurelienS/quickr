package handlers

import (
    "errors"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "quickr/models"
    "quickr/services"
)

func TestHandleRedirect_Success(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()

    // Use a real LinkService with an in-memory fake repository
    repo := &services_fakeRepoForHandlers{ FindByAliasFunc: func(alias string) (*models.Link, error) { return &models.Link{ID: 1, Alias: alias, URL: "https://example.com"}, nil }, IncrementClicksFunc: func(id uint) error { return nil } }
    svc := services.NewLinkService(repo)
    h := &AppHandler{ LinkService: svc }
    r.GET("/:alias", h.HandleRedirect())

    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/foo", nil)
    r.ServeHTTP(w, req)

    if w.Code != http.StatusFound { t.Fatalf("expected 302, got %d", w.Code) }
    if loc := w.Header().Get("Location"); loc != "https://example.com" {
        t.Fatalf("unexpected location: %q", loc)
    }
}

func TestHandleRedirect_NotFound(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    repo := &services_fakeRepoForHandlers{ FindByAliasFunc: func(alias string) (*models.Link, error) { return nil, errors.New("db not found") } }
    svc := services.NewLinkService(repo)
    h := &AppHandler{ LinkService: svc }
    r.GET("/:alias", h.HandleRedirect())

    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/missing", nil)
    r.ServeHTTP(w, req)
    if w.Code != http.StatusNotFound { t.Fatalf("expected 404, got %d", w.Code) }
}

// services_fakeRepoForHandlers provides only the methods used by LinkService in redirect tests
type services_fakeRepoForHandlers struct {
    FindByAliasFunc func(alias string) (*models.Link, error)
    IncrementClicksFunc func(id uint) error
}

func (f *services_fakeRepoForHandlers) Create(link *models.Link) error { return nil }
func (f *services_fakeRepoForHandlers) FindByAlias(alias string) (*models.Link, error) { return f.FindByAliasFunc(alias) }
func (f *services_fakeRepoForHandlers) FindByID(id string) (*models.Link, error) { return nil, errors.New("unused") }
func (f *services_fakeRepoForHandlers) ExistsByAlias(alias string) (bool, error) { return false, nil }
func (f *services_fakeRepoForHandlers) ExistsByAliasExceptID(alias string, id string) (bool, error) { return false, nil }
func (f *services_fakeRepoForHandlers) Delete(link *models.Link) error { return nil }
func (f *services_fakeRepoForHandlers) Save(link *models.Link) error { return nil }
func (f *services_fakeRepoForHandlers) ListAll() ([]models.Link, error) { return nil, nil }
func (f *services_fakeRepoForHandlers) Search(query string) ([]models.Link, error) { return nil, nil }
func (f *services_fakeRepoForHandlers) IncrementClicks(id uint) error { if f.IncrementClicksFunc == nil { return nil }; return f.IncrementClicksFunc(id) }
func (f *services_fakeRepoForHandlers) GetLinkByID(id string) (*models.Link, error) { return nil, errors.New("unused") }


