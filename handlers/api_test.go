package handlers

import (
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/gin-gonic/gin"
    "quickr/models"
    "quickr/services"
)

// in-memory fake repo for API handler tests
type apiFakeRepo struct {
    CreateFunc func(link *models.Link) error
    ExistsByAliasFunc func(alias string) (bool, error)
}

func (f *apiFakeRepo) Create(link *models.Link) error { if f.CreateFunc == nil { return nil }; return f.CreateFunc(link) }
func (f *apiFakeRepo) FindByAlias(alias string) (*models.Link, error) { return nil, errors.New("unused") }
func (f *apiFakeRepo) FindByID(id string) (*models.Link, error) { return nil, errors.New("unused") }
func (f *apiFakeRepo) ExistsByAlias(alias string) (bool, error) { if f.ExistsByAliasFunc == nil { return false, nil }; return f.ExistsByAliasFunc(alias) }
func (f *apiFakeRepo) ExistsByAliasExceptID(alias string, id string) (bool, error) { return false, nil }
func (f *apiFakeRepo) Delete(link *models.Link) error { return nil }
func (f *apiFakeRepo) Save(link *models.Link) error { return nil }
func (f *apiFakeRepo) ListAll() ([]models.Link, error) { return nil, nil }
func (f *apiFakeRepo) Search(query string) ([]models.Link, error) { return nil, nil }
func (f *apiFakeRepo) IncrementClicks(id uint) error { return nil }

func setupRouter(h *AppHandler) *gin.Engine {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.POST("/api/links", h.CreateLink())
    return r
}

func TestCreateLink_JSON_Success(t *testing.T) {
    repo := &apiFakeRepo{ ExistsByAliasFunc: func(alias string) (bool, error) { return false, nil }, CreateFunc: func(link *models.Link) error { return nil } }
    svc := services.NewLinkService(repo)
    h := &AppHandler{ LinkService: svc }
    r := setupRouter(h)

    body := `{"alias":"foo","url":"https://example.com"}`
    req := httptest.NewRequest("POST", "/api/links", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusCreated { t.Fatalf("expected 201, got %d - %s", w.Code, w.Body.String()) }
}

func TestCreateLink_JSON_ValidationErrors(t *testing.T) {
    // invalid body
    svc := services.NewLinkService(&apiFakeRepo{})
    h := &AppHandler{ LinkService: svc }
    r := setupRouter(h)
    req := httptest.NewRequest("POST", "/api/links", strings.NewReader("{"))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest { t.Fatalf("expected 400 for bad body, got %d", w.Code) }

    // alias exists
    repo := &apiFakeRepo{ ExistsByAliasFunc: func(alias string) (bool, error) { return true, nil } }
    svc2 := services.NewLinkService(repo)
    h2 := &AppHandler{ LinkService: svc2 }
    r2 := setupRouter(h2)
    body := `{"alias":"foo","url":"https://example.com"}`
    req2 := httptest.NewRequest("POST", "/api/links", strings.NewReader(body))
    req2.Header.Set("Content-Type", "application/json")
    w2 := httptest.NewRecorder()
    r2.ServeHTTP(w2, req2)
    if w2.Code != http.StatusConflict { t.Fatalf("expected 409 for alias exists, got %d", w2.Code) }
}


