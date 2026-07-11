package service

import (
	"fmt"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"
)

// MercenaryService edits one campaign's descr_mercenaries.txt. Regions is read-only
// reference data (parsed once from descr_regions.txt when the campaign is loaded) used
// by the frontend to tag/filter pools — there is no "create a region" feature.
type MercenaryService struct {
	repo    repository.MercenaryRepository
	regions []string
}

func NewMercenaryService(repo repository.MercenaryRepository, regions []string) *MercenaryService {
	return &MercenaryService{repo: repo, regions: regions}
}

func (s *MercenaryService) GetPools() []domain.MercenaryPool {
	return s.repo.GetAll()
}

func (s *MercenaryService) GetRegions() []string {
	return s.regions
}

func (s *MercenaryService) GetPoolByName(name string) (*domain.MercenaryPool, error) {
	p, ok := s.repo.GetByName(name)
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", name)
	}
	return &p, nil
}

// UpdatePool replaces a pool's regions and unit list wholesale — covers editing a unit's
// fields, adding/removing units, and adding/removing regions in one call, mirroring
// M2TWBuildingService.UpdateBuildingLevel's whole-level-replace pattern.
func (s *MercenaryService) UpdatePool(originalName string, regions []string, units []domain.MercenaryUnit) error {
	p, ok := s.repo.GetByName(originalName)
	if !ok {
		return fmt.Errorf("pool not found: %s", originalName)
	}
	p.Regions = regions
	p.Units = units
	return s.repo.UpdatePool(originalName, p)
}

// Rename changes a pool's name, keeping its regions/units.
func (s *MercenaryService) Rename(originalName, newName string) error {
	if newName == "" {
		return fmt.Errorf("pool name cannot be empty")
	}
	if _, exists := s.repo.GetByName(newName); exists && newName != originalName {
		return fmt.Errorf("pool %q already exists", newName)
	}
	p, ok := s.repo.GetByName(originalName)
	if !ok {
		return fmt.Errorf("pool not found: %s", originalName)
	}
	p.Name = newName
	return s.repo.UpdatePool(originalName, p)
}

func (s *MercenaryService) CreatePool(name string, regions []string) error {
	if name == "" {
		return fmt.Errorf("pool name cannot be empty")
	}
	if _, exists := s.repo.GetByName(name); exists {
		return fmt.Errorf("pool %q already exists", name)
	}
	return s.repo.AddPool(domain.MercenaryPool{Name: name, Regions: regions})
}

func (s *MercenaryService) DeletePool(name string) error {
	return s.repo.DeletePool(name)
}

func (s *MercenaryService) GetChangeType(name string) string {
	ct, ok := s.repo.GetChanges()[name]
	if !ok {
		return "none"
	}
	switch ct {
	case repository.ChangeModified:
		return "modified"
	case repository.ChangeAdded:
		return "added"
	case repository.ChangeDeleted:
		return "deleted"
	default:
		return "none"
	}
}

func (s *MercenaryService) Revert(name string) error {
	return s.repo.RevertPool(name)
}

func (s *MercenaryService) RevertAll() {
	s.repo.RevertAll()
}

func (s *MercenaryService) HasUnsavedChanges() bool {
	return s.repo.HasUnsavedChanges()
}
