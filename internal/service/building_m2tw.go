package service

import (
	"fmt"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"
)

type M2TWBuildingService struct {
	repo repository.M2TWGameRepository
}

func NewM2TWBuildingService(repo repository.M2TWGameRepository) *M2TWBuildingService {
	return &M2TWBuildingService{repo: repo}
}

func (s *M2TWBuildingService) GetFactions() []domain.M2TWFaction {
	return s.repo.GetFactions()
}

func (s *M2TWBuildingService) GetProjectileTypes() []string {
	return s.repo.GetProjectileTypes()
}

func (s *M2TWBuildingService) GetBuildings() []domain.M2TWBuildingGroup {
	return s.repo.GetBuildings()
}

func (s *M2TWBuildingService) GetBuildingByName(name string) (*domain.M2TWBuildingGroup, error) {
	b, ok := s.repo.GetBuildingByName(name)
	if !ok {
		return nil, fmt.Errorf("building not found")
	}
	return &b, nil
}

func (s *M2TWBuildingService) GetUnitBuildings(unitType string) []domain.RecruitLocation {
	return s.repo.GetUnitBuildings(unitType)
}

func (s *M2TWBuildingService) GetFactionBuildings(faction string) []domain.M2TWBuildingGroup {
	return s.repo.GetFactionBuildings(faction)
}

func (s *M2TWBuildingService) GetReligions() []string {
	return s.repo.GetReligions()
}

func (s *M2TWBuildingService) GetHiddenResources() []string {
	return s.repo.GetHiddenResources()
}

func (s *M2TWBuildingService) UpdateBuildingLevel(groupName, levelName string, pools []domain.M2TWRecruitPool, bonusLines []string) error {
	return s.repo.UpdateBuildingLevel(groupName, levelName, pools, bonusLines)
}

func (s *M2TWBuildingService) UpdateBuildingLevelProps(groupName, levelName string, cost, construction, convertTo int, settlementMin, settlementType string, requiredFactions []string, dependencyGroup, dependencyLevel string, upgrades []string) error {
	return s.repo.UpdateBuildingLevelProps(groupName, levelName, cost, construction, convertTo, settlementMin, settlementType, requiredFactions, dependencyGroup, dependencyLevel, upgrades)
}

func (s *M2TWBuildingService) RevertBuildings() error {
	return s.repo.RevertBuildings()
}
