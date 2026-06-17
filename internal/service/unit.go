package service

import (
	"modding-utils/internal/domain"
	"modding-utils/internal/repository"
)

type UnitService struct {
	repo repository.GameRepository
}

func NewUnitService(repo repository.GameRepository) *UnitService {
	return &UnitService{repo: repo}
}

// GetByFaction возвращает юнитов выбранной фракции.
func (s *UnitService) GetByFaction(faction string) []domain.Unit {
	panic("not implemented")
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
