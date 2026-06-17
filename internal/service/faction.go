package service

import (
	"fmt"
	"modding-utils/internal/domain"
	"modding-utils/internal/repository"
)

type FactionService struct {
	repo repository.GameRepository
}

func (s *FactionService) GetFactions() []domain.Faction {
	return s.repo.GetFactions()
}

func (s *FactionService) GetFactionsByName(name string) (*domain.Faction, error) {
	faction, isFind := s.repo.GetFactionByName(name); if !isFind {
		return nil, fmt.Errorf("faction not found")
	}

	return &faction, nil
}