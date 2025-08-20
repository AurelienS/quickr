package repositories

import (
    "errors"

    "gorm.io/gorm"
    "quickr/models"
)

type LinkRepository interface {
    Create(link *models.Link) error
    FindByAlias(alias string) (*models.Link, error)
    FindByID(id string) (*models.Link, error)
    ExistsByAlias(alias string) (bool, error)
    ExistsByAliasExceptID(alias string, id string) (bool, error)
    Delete(link *models.Link) error
    Save(link *models.Link) error
    ListAll() ([]models.Link, error)
    Search(query string) ([]models.Link, error)
    IncrementClicks(id uint) error
}

type GormLinkRepository struct { db *gorm.DB }

func NewGormLinkRepository(db *gorm.DB) *GormLinkRepository { return &GormLinkRepository{db: db} }

func (r *GormLinkRepository) Create(link *models.Link) error { return r.db.Create(link).Error }

func (r *GormLinkRepository) FindByAlias(alias string) (*models.Link, error) {
    var link models.Link
    if err := r.db.Where("alias = ? AND deleted_at IS NULL", alias).First(&link).Error; err != nil {
        return nil, err
    }
    return &link, nil
}

func (r *GormLinkRepository) FindByID(id string) (*models.Link, error) {
    var link models.Link
    if err := r.db.First(&link, id).Error; err != nil {
        return nil, err
    }
    return &link, nil
}

func (r *GormLinkRepository) ExistsByAlias(alias string) (bool, error) {
    var link models.Link
    err := r.db.Where("alias = ?", alias).First(&link).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return false, nil
    }
    return err == nil, err
}

func (r *GormLinkRepository) ExistsByAliasExceptID(alias string, id string) (bool, error) {
    var link models.Link
    err := r.db.Where("alias = ? AND id != ?", alias, id).First(&link).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return false, nil
    }
    return err == nil, err
}

func (r *GormLinkRepository) Delete(link *models.Link) error { return r.db.Delete(link).Error }
func (r *GormLinkRepository) Save(link *models.Link) error   { return r.db.Save(link).Error }

func (r *GormLinkRepository) ListAll() ([]models.Link, error) {
    var links []models.Link
    if err := r.db.Order("created_at desc").Find(&links).Error; err != nil {
        return nil, err
    }
    return links, nil
}

func (r *GormLinkRepository) Search(q string) ([]models.Link, error) {
    var links []models.Link
    like := "%" + q + "%"
    if err := r.db.Where("alias LIKE ? OR url LIKE ?", like, like).Order("created_at desc").Find(&links).Error; err != nil {
        return nil, err
    }
    return links, nil
}

func (r *GormLinkRepository) IncrementClicks(id uint) error {
    return r.db.Model(&models.Link{ID: id}).Update("clicks", gorm.Expr("clicks + ?", 1)).Error
}


