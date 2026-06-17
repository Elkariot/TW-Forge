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

// GetByFaction возвращает юнитов выбранной фракции.
func (s *UnitService) GetByFaction(faction string) ([]domain.Unit, error) {
	units, err := s.repo.GetUnitsByFaction(faction); if err != nil {
		return nil, err
	}

	return units, err
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
	unit, isFind := s.repo.GetUnitByType(unitType); if !isFind {
		return nil, fmt.Errorf("unit not found")
	}

	return &unit, nil
}

// CopyUnit создаёт копию юнита exclusively для одной фракции.
// Новый тип = оригинальный тип + "_" + faction.
func (s *UnitService) CopyUnit(unitType, faction string) error {
	panic("not implemented")
}

// Update сохраняет изменённые характеристики юнита в репозиторий.
func (s *UnitService) Update(unit domain.Unit) error {
	panic("not implemented")
}

// Create добавляет новый юнит на основе шаблона.
func (s *UnitService) Create(unit domain.Unit) error {
	panic("not implemented")
}

// Delete удаляет юнита и его записи из зданий.
func (s *UnitService) Delete(unitType string) error {
	panic("not implemented")
}

// Revert откатывает изменения одного юнита до исходного состояния.
func (s *UnitService) Revert(unitType string) error {
	panic("not implemented")
}
