package service

import (
	"fmt"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"
)

type BuildingService struct {
	repo repository.GameRepository
}

func NewBuildingService(repo repository.GameRepository) *BuildingService {
	return &BuildingService{
		repo: repo,
	}
}

func (s *BuildingService) GetCultureNames() []string {
	return s.repo.GetCultureNames()
}

func (s *BuildingService) GetProjectileTypes() []string {
	return s.repo.GetProjectileTypes()
}

func (s *BuildingService) GetBuildings() []domain.BuildingGroup {
	return s.repo.GetBuildings()
}

func (s *BuildingService) GetBuildingByName(name string) (*domain.BuildingGroup, error) {
	building, isFind := s.repo.GetBuildingByName(name); if !isFind {
		return nil, fmt.Errorf("building not found")
	}

	return &building, nil
}

func (s *BuildingService) GetUnitBuilding(unitType string) []domain.RecruitLocation {
	return s.repo.GetUnitBuildings(unitType)
}

func (s *BuildingService) GetFactionBuildings(faction string) ([]domain.BuildingGroup, error) {
	culture, ok := s.repo.GetFactionCulture(faction)
	if !ok {
		return nil, fmt.Errorf("faction not found")
	}
	return s.repo.GetCultureBuildings(culture), nil
}

func (s *BuildingService) UpdateBuildingLevel(groupName, levelName string, slots []domain.RecruitSlot) error {
	return s.repo.UpdateBuildingLevel(groupName, levelName, slots)
}

func (s *BuildingService) UpdateBuildingLevelProps(groupName, levelName string, cost, construction int, settlementMin string, requiredCultures []string, dependencyGroup, dependencyLevel string, upgrades, bonusLines []string) error {
	return s.repo.UpdateBuildingLevelProps(groupName, levelName, cost, construction, settlementMin, requiredCultures, dependencyGroup, dependencyLevel, upgrades, bonusLines)
}

func (s *BuildingService) RevertBuildings() error {
	return s.repo.RevertBuildings()
}