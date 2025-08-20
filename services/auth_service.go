package services

import (
    "crypto/rand"
    "encoding/base64"
    "errors"
    "fmt"
    "strings"
    "time"

    "quickr/models"
    "quickr/repositories"
)

type Mailer interface { SendMagicLink(email, link string) error }

type AuthService struct {
    users      repositories.UserRepository
    invites    repositories.InvitationRepository
    mailer     Mailer
    buildURL   func(base, token string) string
    appBaseURL string
}

func NewAuthService(users repositories.UserRepository, invites repositories.InvitationRepository, mailer Mailer, appBaseURL string, buildLink func(base, token string) string) *AuthService {
    if buildLink == nil {
        buildLink = func(base, token string) string { return fmt.Sprintf("%s/magic?token=%s", strings.TrimRight(base, "/"), token) }
    }
    return &AuthService{users: users, invites: invites, mailer: mailer, appBaseURL: appBaseURL, buildURL: buildLink}
}

func (a *AuthService) EnsureAdmin(email string) error {
    e := strings.TrimSpace(strings.ToLower(email))
    u, err := a.users.FindByEmail(e)
    if err != nil {
        u = &models.User{Email: e, Role: "admin"}
    }
    u.Role = "admin"
    u.LastLogin = time.Now()
    return a.users.Save(u)
}

func (a *AuthService) generateToken(n int) (string, error) {
    b := make([]byte, n)
    if _, err := rand.Read(b); err != nil { return "", err }
    return base64.RawURLEncoding.EncodeToString(b), nil
}

// CreateMagicLinkInvite creates or replaces a pending invite and sends it; marks sent when success
func (a *AuthService) CreateMagicLinkInvite(email, resolvedBaseURL string) (string, error) {
    e := strings.TrimSpace(strings.ToLower(email))
    _ = a.invites.RevokePendingAndSent(e)
    token, err := a.generateToken(32)
    if err != nil { return "", err }
    inv := &models.Invitation{Email: e, Token: token, Status: "pending", ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}
    if err := a.invites.Create(inv); err != nil { return "", err }
    link := a.buildURL(resolvedBaseURL, token)
    if err := a.mailer.SendMagicLink(e, link); err != nil { return "", err }
    inv.Status = "sent"
    _ = a.invites.Save(inv)
    return token, nil
}

// RequireAndSendMagicLink sends a link only if a previous invite exists
func (a *AuthService) RequireAndSendMagicLink(email, resolvedBaseURL string) error {
    if _, err := a.invites.FindLatestActiveByEmail(email); err != nil { return errors.New("email not invited") }
    _, err := a.CreateMagicLinkInvite(email, resolvedBaseURL)
    return err
}

// RedeemMagicToken validates an invitation token, upserts user, marks invite used
func (a *AuthService) RedeemMagicToken(token string, assignAdmin func(email string) bool) (email string, role string, err error) {
    inv, err := a.invites.FindByToken(token)
    if err != nil { return "", "", errors.New("invalid token") }
    if inv.Status == "used" || inv.Status == "revoked" || inv.ExpiresAt.Before(time.Now()) {
        return "", "", errors.New("token expired or used")
    }
    // Lookup or create user
    u, err := a.users.FindByEmail(inv.Email)
    if err != nil { u = &models.User{Email: inv.Email, Role: "user"} }
    if assignAdmin != nil && assignAdmin(u.Email) { u.Role = "admin" }
    u.LastLogin = time.Now()
    if err := a.users.Save(u); err != nil { return "", "", err }
    now := time.Now()
    inv.Status = "used"
    inv.UsedAt = &now
    _ = a.invites.Save(inv)
    return u.Email, u.Role, nil
}

func (a *AuthService) GetUserByEmail(email string) (*models.User, error) {
    e := strings.TrimSpace(strings.ToLower(email))
    return a.users.FindByEmail(e)
}

func (a *AuthService) IsUserDisabled(email string) (bool, error) {
    u, err := a.GetUserByEmail(email)
    if err != nil {
        return false, nil
    }
    return u.Disabled, nil
}

// Admin dashboard helpers
func (a *AuthService) ListInvitations(limit int) ([]models.Invitation, error) { return a.invites.List(limit) }
func (a *AuthService) SendInvitationByID(id string, resolvedBaseURL string) (*models.Invitation, error) {
    inv, err := a.invites.FindByID(id)
    if err != nil { return nil, err }
    if inv.Status == "used" || inv.Status == "revoked" { return nil, errors.New("cannot send this invite") }
    if inv.Token == "" {
        token, err := a.generateToken(32)
        if err != nil { return nil, err }
        inv.Token = token
        inv.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
        if err := a.invites.Save(inv); err != nil { return nil, err }
    }
    link := a.buildURL(resolvedBaseURL, inv.Token)
    if err := a.mailer.SendMagicLink(inv.Email, link); err != nil { return nil, err }
    inv.Status = "sent"
    if err := a.invites.Save(inv); err != nil { return nil, err }
    return inv, nil
}
func (a *AuthService) RevokeInvitationByID(id string) (*models.Invitation, error) {
    inv, err := a.invites.FindByID(id)
    if err != nil { return nil, err }
    if inv.Status == "used" || inv.Status == "revoked" { return nil, errors.New("cannot revoke this invite") }
    inv.Status = "revoked"
    if err := a.invites.Save(inv); err != nil { return nil, err }
    return inv, nil
}
func (a *AuthService) RevokeAllForEmail(email string) error { return a.invites.RevokeAllByEmail(strings.ToLower(strings.TrimSpace(email))) }

func (a *AuthService) DisableUser(email string) error {
    e := strings.ToLower(strings.TrimSpace(email))
    u, err := a.users.FindByEmail(e)
    if err != nil {
        // create disabled user if not exists
        u = &models.User{Email: e, Role: "user", Disabled: true}
        return a.users.Create(u)
    }
    if u.Disabled {
        return errors.New("user already revoked")
    }
    u.Disabled = true
    return a.users.Save(u)
}

// AnnotateInvites joins invitations with user disabled state for UI rendering
func (a *AuthService) AnnotateInvitesWithUserDisabled(invites []models.Invitation) ([]struct{ Inv models.Invitation; UserDisabled bool }, error) {
    emailSet := map[string]struct{}{}
    for _, inv := range invites { emailSet[inv.Email] = struct{}{} }
    emails := make([]string, 0, len(emailSet))
    for e := range emailSet { emails = append(emails, e) }
    users, err := a.users.ListByEmails(emails)
    if err != nil { return nil, err }
    disabled := map[string]bool{}
    for _, u := range users { disabled[u.Email] = u.Disabled }
    out := make([]struct{ Inv models.Invitation; UserDisabled bool }, 0, len(invites))
    for _, inv := range invites { out = append(out, struct{ Inv models.Invitation; UserDisabled bool }{Inv: inv, UserDisabled: disabled[inv.Email]}) }
    return out, nil
}


