package repositories

import (
    "strings"

    "gorm.io/gorm"
    "quickr/models"
)

type UserRepository interface {
    FindByEmail(email string) (*models.User, error)
    Save(user *models.User) error
    Create(user *models.User) error
    ListByEmails(emails []string) ([]models.User, error)
}

type GormUserRepository struct { db *gorm.DB }

func NewGormUserRepository(db *gorm.DB) *GormUserRepository { return &GormUserRepository{db: db} }

func (r *GormUserRepository) FindByEmail(email string) (*models.User, error) {
    var u models.User
    e := strings.TrimSpace(strings.ToLower(email))
    if err := r.db.Where("email = ?", e).First(&u).Error; err != nil { return nil, err }
    return &u, nil
}

func (r *GormUserRepository) Save(user *models.User) error { return r.db.Save(user).Error }
func (r *GormUserRepository) Create(user *models.User) error { return r.db.Create(user).Error }

func (r *GormUserRepository) ListByEmails(emails []string) ([]models.User, error) {
    var users []models.User
    if len(emails) == 0 { return users, nil }
    if err := r.db.Where("email IN ?", emails).Find(&users).Error; err != nil { return nil, err }
    return users, nil
}

var ErrNotFound = gorm.ErrRecordNotFound


