package services

import (
    "quickr/models"
)

// fakeRepo is a function-backed test double for repositories.LinkRepository.
// Any method that is called without its corresponding Func set will panic,
// which helps catch unexpected interactions in tests.
type fakeRepo struct {
    CreateFunc                 func(link *models.Link) error
    FindByAliasFunc            func(alias string) (*models.Link, error)
    FindByIDFunc               func(id string) (*models.Link, error)
    ExistsByAliasFunc          func(alias string) (bool, error)
    ExistsByAliasExceptIDFunc  func(alias string, id string) (bool, error)
    DeleteFunc                 func(link *models.Link) error
    SaveFunc                   func(link *models.Link) error
    ListAllFunc                func() ([]models.Link, error)
    SearchFunc                 func(query string) ([]models.Link, error)
    IncrementClicksFunc        func(id uint) error
}

func (f *fakeRepo) Create(link *models.Link) error {
    if f.CreateFunc == nil { panic("unexpected call to Create") }
    return f.CreateFunc(link)
}

func (f *fakeRepo) FindByAlias(alias string) (*models.Link, error) {
    if f.FindByAliasFunc == nil { panic("unexpected call to FindByAlias") }
    return f.FindByAliasFunc(alias)
}

func (f *fakeRepo) FindByID(id string) (*models.Link, error) {
    if f.FindByIDFunc == nil { panic("unexpected call to FindByID") }
    return f.FindByIDFunc(id)
}

func (f *fakeRepo) ExistsByAlias(alias string) (bool, error) {
    if f.ExistsByAliasFunc == nil { panic("unexpected call to ExistsByAlias") }
    return f.ExistsByAliasFunc(alias)
}

func (f *fakeRepo) ExistsByAliasExceptID(alias string, id string) (bool, error) {
    if f.ExistsByAliasExceptIDFunc == nil { panic("unexpected call to ExistsByAliasExceptID") }
    return f.ExistsByAliasExceptIDFunc(alias, id)
}

func (f *fakeRepo) Delete(link *models.Link) error {
    if f.DeleteFunc == nil { panic("unexpected call to Delete") }
    return f.DeleteFunc(link)
}

func (f *fakeRepo) Save(link *models.Link) error {
    if f.SaveFunc == nil { panic("unexpected call to Save") }
    return f.SaveFunc(link)
}

func (f *fakeRepo) ListAll() ([]models.Link, error) {
    if f.ListAllFunc == nil { panic("unexpected call to ListAll") }
    return f.ListAllFunc()
}

func (f *fakeRepo) Search(query string) ([]models.Link, error) {
    if f.SearchFunc == nil { panic("unexpected call to Search") }
    return f.SearchFunc(query)
}

func (f *fakeRepo) IncrementClicks(id uint) error {
    if f.IncrementClicksFunc == nil { panic("unexpected call to IncrementClicks") }
    return f.IncrementClicksFunc(id)
}


