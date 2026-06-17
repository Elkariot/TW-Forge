package service

import (
	"fmt"
	"modding-utils/internal/domain"
	"modding-utils/internal/repository"
)

type UnitService struct {
	repo repository.GameRepository
}

func NewUnitService(repo repository.GameRepository) *UnitService {
	return &UnitService{repo: repo}
}

func (s *UnitService) GetFactions() []domain.Faction {
	return s.repo.GetFactions()
}

func (s *UnitService) GetByFaction(faction string) ([]domain.Unit, error) {
	return s.repo.GetUnitsByFaction(faction)
}

func (s *UnitService) GetFactionUnitsByCategoryAndClass(faction string) (map[string]map[string][]domain.Unit, error) {
	units, err := s.repo.GetUnitsByFaction(faction)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string][]domain.Unit)
	for _, unit := range units {
		if _, ok := result[unit.Category]; !ok {
			result[unit.Category] = make(map[string][]domain.Unit)
		}
		result[unit.Category][unit.Class] = append(result[unit.Category][unit.Class], unit)
	}

	return result, nil
}

func (s *UnitService) GetUnitByType(unitType string) (*domain.Unit, error) {
	unit, ok := s.repo.GetUnitByType(unitType)
	if !ok {
		return nil, fmt.Errorf("unit not found")
	}
	return &unit, nil
}

func (s *UnitService) Update(originalType string, unit domain.Unit) error {
	if err := s.repo.UpdateUnit(originalType, unit); err != nil {
		return err
	}
	return s.repo.SaveDraft()
}

func (s *UnitService) Create(unit domain.Unit) error {
	if err := s.repo.AddUnit(unit); err != nil {
		return err
	}
	return s.repo.SaveDraft()
}

func (s *UnitService) Delete(unitType string) error {
	if err := s.repo.DeleteUnit(unitType); err != nil {
		return err
	}
	return s.repo.SaveDraft()
}

func (s *UnitService) CopyUnit(unitType, faction string) error {
	panic("not implemented")
}

func (s *UnitService) Revert(unitType string) error {
	return s.repo.RevertUnit(unitType)
}

func (s *UnitService) RevertAll() {
	s.repo.RevertAll()
}

func (s *UnitService) HasUnsavedChanges() bool {
	return s.repo.HasUnsavedChanges()
}

func (s *UnitService) Save() error {
	return s.repo.Save()
}
