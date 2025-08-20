package repositories

import (
    "time"

    "gorm.io/gorm"
    "quickr/models"
)

type InvitationRepository interface {
    FindLatestActiveByEmail(email string) (*models.Invitation, error)
    RevokePendingAndSent(email string) error
    Create(inv *models.Invitation) error
    Save(inv *models.Invitation) error
    FindByToken(token string) (*models.Invitation, error)
    FindByID(id string) (*models.Invitation, error)
    List(limit int) ([]models.Invitation, error)
    RevokeByID(id string) error
    RevokeAllByEmail(email string) error
}

type GormInvitationRepository struct { db *gorm.DB }

func NewGormInvitationRepository(db *gorm.DB) *GormInvitationRepository { return &GormInvitationRepository{db: db} }

func (r *GormInvitationRepository) FindLatestActiveByEmail(email string) (*models.Invitation, error) {
    var inv models.Invitation
    if err := r.db.Where("email = ? AND status <> ?", email, "revoked").Order("created_at desc").First(&inv).Error; err != nil {
        return nil, err
    }
    return &inv, nil
}

func (r *GormInvitationRepository) RevokePendingAndSent(email string) error {
    return r.db.Model(&models.Invitation{}).Where("email = ? AND status IN ?", email, []string{"pending", "sent"}).Update("status", "revoked").Error
}

func (r *GormInvitationRepository) Create(inv *models.Invitation) error { return r.db.Create(inv).Error }
func (r *GormInvitationRepository) Save(inv *models.Invitation) error   { return r.db.Save(inv).Error }

func (r *GormInvitationRepository) FindByToken(token string) (*models.Invitation, error) {
    var inv models.Invitation
    if err := r.db.Where("token = ?", token).First(&inv).Error; err != nil { return nil, err }
    return &inv, nil
}

func DefaultExpiry() time.Time { return time.Now().Add(7 * 24 * time.Hour) }

func (r *GormInvitationRepository) FindByID(id string) (*models.Invitation, error) {
    var inv models.Invitation
    if err := r.db.First(&inv, id).Error; err != nil { return nil, err }
    return &inv, nil
}

func (r *GormInvitationRepository) List(limit int) ([]models.Invitation, error) {
    var invites []models.Invitation
    tx := r.db.Order("created_at desc")
    if limit > 0 { tx = tx.Limit(limit) }
    if err := tx.Find(&invites).Error; err != nil { return nil, err }
    return invites, nil
}

func (r *GormInvitationRepository) RevokeByID(id string) error {
    return r.db.Model(&models.Invitation{}).Where("id = ?", id).Update("status", "revoked").Error
}

func (r *GormInvitationRepository) RevokeAllByEmail(email string) error {
    return r.db.Model(&models.Invitation{}).Where("email = ? AND status <> ?", email, "revoked").Update("status", "revoked").Error
}


