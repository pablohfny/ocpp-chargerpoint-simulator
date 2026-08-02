package services

import (
	"errors"
	"sort"
	"sync"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/infrastructure/persistence"
)

// ErrPartnerNotFound is returned when no profile matches a slug.
var ErrPartnerNotFound = errors.New("partner not found")

// ErrPartnerExists is returned when creating a profile with a taken slug.
var ErrPartnerExists = errors.New("partner already exists")

// OCPIPartnerService owns the OCPI partner profiles and their persistence.
type OCPIPartnerService struct {
	mu       sync.RWMutex
	partners map[string]entities.OCPIPartner
	store    *persistence.OCPIPartnerStore
}

// NewOCPIPartnerService creates a partner service backed by the given store.
func NewOCPIPartnerService(store *persistence.OCPIPartnerStore) *OCPIPartnerService {
	return &OCPIPartnerService{
		partners: make(map[string]entities.OCPIPartner),
		store:    store,
	}
}

// Load reads the persisted profiles into memory.
func (s *OCPIPartnerService) Load() error {
	if s.store == nil {
		return nil
	}

	partners, err := s.store.Load()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, partner := range partners {
		s.partners[partner.Slug] = partner
	}
	return nil
}

// List returns every profile ordered by slug.
func (s *OCPIPartnerService) List() []entities.OCPIPartner {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]entities.OCPIPartner, 0, len(s.partners))
	for _, partner := range s.partners {
		result = append(result, partner)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Slug < result[j].Slug })
	return result
}

// Get returns a single profile by slug.
func (s *OCPIPartnerService) Get(slug string) (entities.OCPIPartner, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	partner, ok := s.partners[slug]
	if !ok {
		return entities.OCPIPartner{}, ErrPartnerNotFound
	}
	return partner, nil
}

// Create validates and stores a new profile.
func (s *OCPIPartnerService) Create(partner entities.OCPIPartner) (entities.OCPIPartner, error) {
	partner.Normalize()
	if err := partner.Validate(); err != nil {
		return entities.OCPIPartner{}, err
	}

	s.mu.Lock()
	if _, exists := s.partners[partner.Slug]; exists {
		s.mu.Unlock()
		return entities.OCPIPartner{}, ErrPartnerExists
	}
	s.partners[partner.Slug] = partner
	s.mu.Unlock()

	if err := s.persist(); err != nil {
		return entities.OCPIPartner{}, err
	}
	return partner, nil
}

// Update replaces an existing profile. The slug in the path wins over the body.
func (s *OCPIPartnerService) Update(slug string, partner entities.OCPIPartner) (entities.OCPIPartner, error) {
	partner.Slug = slug
	partner.Normalize()
	if err := partner.Validate(); err != nil {
		return entities.OCPIPartner{}, err
	}

	s.mu.Lock()
	if _, exists := s.partners[partner.Slug]; !exists {
		s.mu.Unlock()
		return entities.OCPIPartner{}, ErrPartnerNotFound
	}
	s.partners[partner.Slug] = partner
	s.mu.Unlock()

	if err := s.persist(); err != nil {
		return entities.OCPIPartner{}, err
	}
	return partner, nil
}

// Delete removes a profile.
func (s *OCPIPartnerService) Delete(slug string) error {
	s.mu.Lock()
	if _, exists := s.partners[slug]; !exists {
		s.mu.Unlock()
		return ErrPartnerNotFound
	}
	delete(s.partners, slug)
	s.mu.Unlock()

	return s.persist()
}

// persist writes the current profiles to disk.
func (s *OCPIPartnerService) persist() error {
	if s.store == nil {
		return nil
	}
	return s.store.Save(s.List())
}
