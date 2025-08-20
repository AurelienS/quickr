package services

import (
    "errors"
    "testing"

    "quickr/models"
)

// fakeRepo is defined in services/testhelpers_test.go for reuse across tests.

func TestCreateLink_Success(t *testing.T) {
    var created *models.Link
    repo := &fakeRepo{
        ExistsByAliasFunc: func(alias string) (bool, error) { if alias != "foo" { t.Fatalf("expected alias 'foo', got %q", alias) }; return false, nil },
        CreateFunc: func(link *models.Link) error { created = link; return nil },
    }
    svc := NewLinkService(repo)

    link, err := svc.CreateLink("foo", "https://example.com", "alice@example.com")
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if link == nil { t.Fatalf("expected link, got nil") }
    if link.Alias != "foo" { t.Errorf("alias mismatch: %q", link.Alias) }
    if link.URL != "https://example.com" { t.Errorf("url mismatch: %q", link.URL) }
    if link.CreatorName != "alice@example.com" { t.Errorf("creator mismatch: %q", link.CreatorName) }
    if created == nil || created != link { t.Fatalf("expected repo.Create to receive the link instance") }
}

func TestCreateLink_TrimsInputs(t *testing.T) {
    repo := &fakeRepo{
        ExistsByAliasFunc: func(alias string) (bool, error) {
            if alias != "foo" { t.Fatalf("expected trimmed alias 'foo', got %q", alias) }
            return false, nil
        },
        CreateFunc: func(link *models.Link) error {
            if link.Alias != "foo" { t.Fatalf("expected trimmed alias, got %q", link.Alias) }
            if link.URL != "https://example.com" { t.Fatalf("expected trimmed url, got %q", link.URL) }
            return nil
        },
    }
    svc := NewLinkService(repo)

    _, err := svc.CreateLink("  foo  ", "  https://example.com  ", "alice")
    if err != nil { t.Fatalf("unexpected error: %v", err) }
}

func TestCreateLink_MissingFields(t *testing.T) {
    svc := NewLinkService(&fakeRepo{})
    if _, err := svc.CreateLink("", "https://example.com", "alice"); err == nil {
        t.Fatalf("expected error for missing alias")
    }
    if _, err := svc.CreateLink("foo", "", "alice"); err == nil {
        t.Fatalf("expected error for missing url")
    }
}

func TestCreateLink_ReservedAlias(t *testing.T) {
    // 'admin' is reserved per domain/reserved
    svc := NewLinkService(&fakeRepo{})
    _, err := svc.CreateLink("admin", "https://example.com", "alice")
    if !errors.Is(err, ErrAliasReserved) {
        t.Fatalf("expected ErrAliasReserved, got %v", err)
    }
}

func TestCreateLink_InvalidURL(t *testing.T) {
    svc := NewLinkService(&fakeRepo{})
    _, err := svc.CreateLink("foo", "nota-valid-url", "alice")
    if !errors.Is(err, ErrInvalidURL) {
        t.Fatalf("expected ErrInvalidURL, got %v", err)
    }
}

func TestCreateLink_AliasExists(t *testing.T) {
    repo := &fakeRepo{
        ExistsByAliasFunc: func(alias string) (bool, error) { return true, nil },
    }
    svc := NewLinkService(repo)
    _, err := svc.CreateLink("foo", "https://example.com", "alice")
    if !errors.Is(err, ErrAliasExists) {
        t.Fatalf("expected ErrAliasExists, got %v", err)
    }
}

func TestCreateLink_RepoErrorsPropagate(t *testing.T) {
    someErr := errors.New("boom")
    repo := &fakeRepo{
        ExistsByAliasFunc: func(alias string) (bool, error) { return false, nil },
        CreateFunc: func(link *models.Link) error { return someErr },
    }
    svc := NewLinkService(repo)
    _, err := svc.CreateLink("foo", "https://example.com", "alice")
    if !errors.Is(err, someErr) {
        t.Fatalf("expected repo error propagated, got %v", err)
    }

    repo2 := &fakeRepo{ ExistsByAliasFunc: func(alias string) (bool, error) { return false, someErr } }
    svc2 := NewLinkService(repo2)
    _, err = svc2.CreateLink("foo", "https://example.com", "alice")
    if !errors.Is(err, someErr) {
        t.Fatalf("expected exists error propagated, got %v", err)
    }
}

func TestUpdateLink_Success_AllFields(t *testing.T) {
    saved := false
    repo := &fakeRepo{
        FindByIDFunc: func(id string) (*models.Link, error) {
            if id != "1" { t.Fatalf("expected id '1', got %q", id) }
            return &models.Link{ID: 1, Alias: "foo", URL: "https://old.com", CreatorName: "alice"}, nil
        },
        ExistsByAliasExceptIDFunc: func(alias string, id string) (bool, error) {
            if alias != "bar" || id != "1" { t.Fatalf("unexpected ExistsByAliasExceptID args: %q, %q", alias, id) }
            return false, nil
        },
        SaveFunc: func(link *models.Link) error {
            saved = true
            if link.Alias != "bar" { t.Fatalf("alias not updated: %q", link.Alias) }
            if link.URL != "https://new.com" { t.Fatalf("url not updated: %q", link.URL) }
            if link.CreatorName != "bob" { t.Fatalf("editor not updated: %q", link.CreatorName) }
            return nil
        },
    }
    svc := NewLinkService(repo)

    link, err := svc.UpdateLink("1", "bar", "https://new.com", "bob")
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if !saved { t.Fatalf("expected Save to be called") }
    if link.Alias != "bar" || link.URL != "https://new.com" || link.CreatorName != "bob" {
        t.Fatalf("returned link not updated correctly")
    }
}

func TestUpdateLink_NoChanges(t *testing.T) {
    repo := &fakeRepo{
        FindByIDFunc: func(id string) (*models.Link, error) { return &models.Link{ID: 2, Alias: "keep", URL: "https://same.com", CreatorName: "alice"}, nil },
        SaveFunc: func(link *models.Link) error { return nil },
    }
    svc := NewLinkService(repo)

    link, err := svc.UpdateLink("2", "", "", "")
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if link.Alias != "keep" || link.URL != "https://same.com" || link.CreatorName != "alice" {
        t.Fatalf("link should be unchanged when no inputs provided")
    }
}

func TestUpdateLink_ReservedAlias(t *testing.T) {
    repo := &fakeRepo{ FindByIDFunc: func(id string) (*models.Link, error) { return &models.Link{ID: 3, Alias: "foo", URL: "https://x", CreatorName: "a"}, nil } }
    svc := NewLinkService(repo)
    _, err := svc.UpdateLink("3", "admin", "", "") // reserved
    if !errors.Is(err, ErrAliasReserved) {
        t.Fatalf("expected ErrAliasReserved, got %v", err)
    }
}

func TestUpdateLink_AliasExists(t *testing.T) {
    repo := &fakeRepo{
        FindByIDFunc: func(id string) (*models.Link, error) { return &models.Link{ID: 4, Alias: "foo", URL: "https://x", CreatorName: "a"}, nil },
        ExistsByAliasExceptIDFunc: func(alias string, id string) (bool, error) { return true, nil },
    }
    svc := NewLinkService(repo)
    _, err := svc.UpdateLink("4", "taken", "", "")
    if !errors.Is(err, ErrAliasExists) {
        t.Fatalf("expected ErrAliasExists, got %v", err)
    }
}

func TestUpdateLink_InvalidURL(t *testing.T) {
    repo := &fakeRepo{ FindByIDFunc: func(id string) (*models.Link, error) { return &models.Link{ID: 5, Alias: "foo", URL: "https://x", CreatorName: "a"}, nil } }
    svc := NewLinkService(repo)
    _, err := svc.UpdateLink("5", "", "notaurl", "")
    if !errors.Is(err, ErrInvalidURL) {
        t.Fatalf("expected ErrInvalidURL, got %v", err)
    }
}

func TestUpdateLink_FindOrSaveErrors(t *testing.T) {
    findErr := errors.New("missing")
    repo := &fakeRepo{ FindByIDFunc: func(id string) (*models.Link, error) { return nil, findErr } }
    svc := NewLinkService(repo)
    if _, err := svc.UpdateLink("404", "", "", ""); !errors.Is(err, ErrLinkNotFound) {
        t.Fatalf("expected ErrLinkNotFound, got %v", err)
    }

    saveErr := errors.New("save fail")
    repo2 := &fakeRepo{
        FindByIDFunc: func(id string) (*models.Link, error) { return &models.Link{ID: 6, Alias: "a", URL: "https://x", CreatorName: "a"}, nil },
        SaveFunc: func(link *models.Link) error { return saveErr },
    }
    svc2 := NewLinkService(repo2)
    if _, err := svc2.UpdateLink("6", "", "", ""); !errors.Is(err, saveErr) {
        t.Fatalf("expected save error to propagate, got %v", err)
    }
}

func TestDeleteLink_Success(t *testing.T) {
    toDelete := &models.Link{ID: 10, Alias: "foo"}
    deleted := false
    repo := &fakeRepo{
        FindByIDFunc: func(id string) (*models.Link, error) { return toDelete, nil },
        DeleteFunc:    func(link *models.Link) error { if link != toDelete { t.Fatalf("unexpected link in delete") }; deleted = true; return nil },
    }
    svc := NewLinkService(repo)
    link, err := svc.DeleteLink("10")
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if link != toDelete { t.Fatalf("expected returned link to be deleted one") }
    if !deleted { t.Fatalf("expected Delete to be called") }
}

func TestDeleteLink_Errors(t *testing.T) {
    repo := &fakeRepo{ FindByIDFunc: func(id string) (*models.Link, error) { return nil, errors.New("nope") } }
    svc := NewLinkService(repo)
    if _, err := svc.DeleteLink("x"); !errors.Is(err, ErrLinkNotFound) {
        t.Fatalf("expected ErrLinkNotFound, got %v", err)
    }

    repo2 := &fakeRepo{
        FindByIDFunc: func(id string) (*models.Link, error) { return &models.Link{ID: 11, Alias: "a"}, nil },
        DeleteFunc:    func(link *models.Link) error { return errors.New("db down") },
    }
    svc2 := NewLinkService(repo2)
    if _, err := svc2.DeleteLink("11"); err == nil {
        t.Fatalf("expected delete error")
    }
}

func TestListAndSearch(t *testing.T) {
    repo := &fakeRepo{
        ListAllFunc: func() ([]models.Link, error) { return []models.Link{{Alias: "a"}, {Alias: "b"}}, nil },
        SearchFunc:   func(query string) ([]models.Link, error) { if query != "foo" { t.Fatalf("expected query 'foo'") }; return []models.Link{{Alias: "foo"}}, nil },
    }
    svc := NewLinkService(repo)

    list, err := svc.ListLinks()
    if err != nil || len(list) != 2 { t.Fatalf("expected 2 links, err=%v", err) }

    results, err := svc.SearchLinks("foo")
    if err != nil || len(results) != 1 || results[0].Alias != "foo" { t.Fatalf("unexpected search results: %+v, err=%v", results, err) }
}

func TestFindByAlias(t *testing.T) {
    repo := &fakeRepo{ FindByAliasFunc: func(alias string) (*models.Link, error) { return &models.Link{Alias: alias}, nil } }
    svc := NewLinkService(repo)
    link, err := svc.FindByAlias("foo")
    if err != nil || link.Alias != "foo" { t.Fatalf("unexpected result: link=%+v err=%v", link, err) }

    repo2 := &fakeRepo{ FindByAliasFunc: func(alias string) (*models.Link, error) { return nil, errors.New("not found in db") } }
    svc2 := NewLinkService(repo2)
    if _, err := svc2.FindByAlias("missing"); err == nil || err.Error() != "not found" {
        t.Fatalf("expected generic not found error, got %v", err)
    }
}

func TestIncrementClicks_And_GetLinkByID(t *testing.T) {
    incrCalled := false
    repo := &fakeRepo{
        IncrementClicksFunc: func(id uint) error { if id != 42 { t.Fatalf("expected id 42, got %d", id) }; incrCalled = true; return nil },
        FindByIDFunc:        func(id string) (*models.Link, error) { if id != "7" { t.Fatalf("expected id '7'") }; return &models.Link{ID: 7, Alias: "a"}, nil },
    }
    svc := NewLinkService(repo)
    if err := svc.IncrementClicks(42); err != nil || !incrCalled { t.Fatalf("increment clicks failed: %v", err) }

    link, err := svc.GetLinkByID("7")
    if err != nil || link.ID != 7 { t.Fatalf("unexpected get by id: link=%+v err=%v", link, err) }

    repo2 := &fakeRepo{ FindByIDFunc: func(id string) (*models.Link, error) { return nil, errors.New("no record") } }
    svc2 := NewLinkService(repo2)
    if _, err := svc2.GetLinkByID("missing"); err == nil || err.Error() != "link not found" {
        t.Fatalf("expected 'link not found' error, got %v", err)
    }
}


