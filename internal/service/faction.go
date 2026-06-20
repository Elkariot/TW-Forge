package service

import (
	"fmt"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"
)

type FactionService struct {
	repo repository.GameRepository
}

func NewFactionService(repo repository.GameRepository) *FactionService {
	return &FactionService{
		repo: repo,
	}
}

func (s *FactionService) GetFactions() []domain.Faction {
	return s.repo.GetFactions()
}

func (s *FactionService) GetFactionByName(name string) (*domain.Faction, error) {
	faction, ok := s.repo.GetFactionByName(name)
	if !ok {
		return nil, fmt.Errorf("faction not found")
	}
	return &faction, nil
}

func (s *FactionService) GetFactionCulture(name string) (string, error) {
	culture, ok := s.repo.GetFactionCulture(name)
	if !ok {
		return "", fmt.Errorf("faction not found")
	}
	return culture, nil
}