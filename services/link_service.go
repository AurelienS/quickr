package services

import (
    "errors"
    "strings"

    "quickr/domain/reserved"
    "quickr/domain/validation"
    "quickr/models"
    "quickr/repositories"
)

type LinkService struct { repo repositories.LinkRepository }

func NewLinkService(repo repositories.LinkRepository) *LinkService { return &LinkService{repo: repo} }

func (s *LinkService) ValidateURL(urlStr string) bool { return validation.IsValidHTTPURL(urlStr) }

func (s *LinkService) IsAliasReserved(alias string) bool { return reserved.IsReservedAlias(alias) }

func (s *LinkService) CreateLink(alias, targetURL, creator string) (*models.Link, error) {
    alias = strings.TrimSpace(alias)
    targetURL = strings.TrimSpace(targetURL)
    if alias == "" || targetURL == "" {
        return nil, errors.New("alias and url are required")
    }
    if s.IsAliasReserved(alias) {
        return nil, errors.New("alias is reserved")
    }
    if !s.ValidateURL(targetURL) {
        return nil, errors.New("invalid url format")
    }
    if exists, err := s.repo.ExistsByAlias(alias); err != nil { return nil, err } else if exists { return nil, errors.New("alias already exists") }
    link := &models.Link{Alias: alias, URL: targetURL, CreatorName: creator}
    if err := s.repo.Create(link); err != nil { return nil, err }
    return link, nil
}

func (s *LinkService) UpdateLink(id string, newAlias, newURL, editor string) (*models.Link, error) {
    link, err := s.repo.FindByID(id)
    if err != nil { return nil, errors.New("link not found") }
    if newAlias != "" {
        if s.IsAliasReserved(newAlias) {
            return nil, errors.New("alias is reserved")
        }
        if exists, err := s.repo.ExistsByAliasExceptID(newAlias, id); err != nil { return nil, err } else if exists { return nil, errors.New("alias already exists") }
        link.Alias = newAlias
    }
    if newURL != "" {
        if !s.ValidateURL(newURL) {
            return nil, errors.New("invalid url format")
        }
        link.URL = newURL
    }
    if editor != "" {
        link.CreatorName = editor
    }
    if err := s.repo.Save(link); err != nil { return nil, err }
    return link, nil
}

func (s *LinkService) DeleteLink(id string) (*models.Link, error) {
    link, err := s.repo.FindByID(id)
    if err != nil { return nil, errors.New("link not found") }
    if err := s.repo.Delete(link); err != nil { return nil, err }
    return link, nil
}

func (s *LinkService) ListLinks() ([]models.Link, error) {
    return s.repo.ListAll()
}

func (s *LinkService) SearchLinks(query string) ([]models.Link, error) {
    return s.repo.Search(query)
}

func (s *LinkService) FindByAlias(alias string) (*models.Link, error) {
    link, err := s.repo.FindByAlias(alias)
    if err != nil { return nil, errors.New("not found") }
    return link, nil
}

func (s *LinkService) IncrementClicks(id uint) error { return s.repo.IncrementClicks(id) }

func (s *LinkService) GetLinkByID(id string) (*models.Link, error) {
	link, err := s.repo.FindByID(id)
	if err != nil { return nil, errors.New("link not found") }
	return link, nil
}


