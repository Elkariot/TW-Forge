package service

import (
	"fmt"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"
	"strings"
)

type UnitService struct {
	repo repository.GameRepository
}

func NewUnitService(repo repository.GameRepository) *UnitService {
	return &UnitService{repo: repo}
}

func (s *UnitService) GetAllUnits() []domain.Unit {
	return s.repo.GetAllUnits()
}

func (s *UnitService) GetDeletedUnits() []domain.Unit {
	return s.repo.GetDeletedUnits()
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
	if _, exists := s.repo.GetUnitByType(unit.Type); exists {
		return fmt.Errorf("unit type already in use: %s", unit.Type)
	}

	if err := s.repo.AddUnit(unit); err != nil {
		return err
	}
	return s.repo.SaveDraft()
}

func (s *UnitService) Delete(unitType string) error {
	if _, exists := s.repo.GetUnitByType(unitType); !exists {
		return fmt.Errorf("unit type not found: %s", unitType)
	}
	if err := s.repo.DeleteUnit(unitType); err != nil {
		return err
	}
	if err := s.repo.SaveDraft(); err != nil {
		return err
	}
	return s.repo.SaveBuildingsDraft()
}

func (s *UnitService) CopyUnit(unitType, faction string) (string, error) {
	unit, ok := s.repo.GetUnitByType(unitType)
	if !ok {
		return "", fmt.Errorf("unit not found")
	}

	base := fmt.Sprintf("%s copy", unit.Type)
	newUnitType := base
	for i := 2; ; i++ {
		if _, exists := s.repo.GetUnitByType(newUnitType); !exists {
			break
		}
		newUnitType = fmt.Sprintf("%s %d", base, i)
	}

	unit.Type = newUnitType
	unit.Dictionary = strings.ReplaceAll(newUnitType, " ", "_")
	unit.Ownership = []string{faction}

	if err := s.repo.AddUnit(unit); err != nil {
		return "", fmt.Errorf("error copying unit: %w", err)
	}

	return newUnitType, s.repo.SaveDraft()
}

func (s *UnitService) CreateUnit(templateType, newType, faction string) error {
	if _, exists := s.repo.GetUnitByType(newType); exists {
		return fmt.Errorf("юнит с типом %q уже существует", newType)
	}
	unit, ok := s.repo.GetUnitByType(templateType)
	if !ok {
		return fmt.Errorf("шаблон %q не найден", templateType)
	}
	unit.Type = newType
	unit.Dictionary = strings.ReplaceAll(newType, " ", "_")
	unit.Ownership = []string{faction}
	if err := s.repo.AddUnit(unit); err != nil {
		return fmt.Errorf("ошибка создания юнита: %w", err)
	}
	return s.repo.SaveDraft()
}

func (s *UnitService) Revert(unitType string) error {
	return s.repo.RevertUnit(unitType)
}

func (s *UnitService) RevertAll() {
	s.repo.RevertAll()
}

func (s *UnitService) GetUnitChangeType(unitType string) string {
	ct, ok := s.repo.GetChanges()[unitType]
	if !ok {
		return "none"
	}
	switch ct {
	case repository.ChangeModified:
		// Отличаем мягкое удаление от обычной правки
		if u, exists := s.repo.GetUnitByType(unitType); exists && u.IsDeleted {
			return "deleted"
		}
		return "modified"
	case repository.ChangeAdded:
		return "added"
	case repository.ChangeDeleted:
		return "deleted"
	default:
		return "none"
	}
}

// Validate проверяет что юнит готов к сохранению.
// originalType — тип до редактирования (исключается из проверки уникальности).
func (s *UnitService) Validate(unit domain.Unit, originalType string) []string {
	var errs []string

	trimmedType := strings.TrimSpace(unit.Type)
	if trimmedType == "" {
		errs = append(errs, "не задан внутренний тип юнита (type)")
	} else if trimmedType != originalType {
		if _, exists := s.repo.GetUnitByType(trimmedType); exists {
			errs = append(errs, fmt.Sprintf("тип \"%s\" уже занят другим юнитом", trimmedType))
		}
	}

	if strings.TrimSpace(unit.Name) == "" {
		errs = append(errs, "не задано отображаемое имя юнита")
	}

	if len(unit.Ownership) == 0 {
		errs = append(errs, "юнит не принадлежит ни одной фракции")
	} else {
		for _, faction := range unit.Ownership {
			if _, ok := s.repo.GetFactionByName(faction); !ok {
				errs = append(errs, fmt.Sprintf("фракция \"%s\" не найдена", faction))
			}
		}
	}

	if len(s.repo.GetUnitBuildings(unit.Type)) == 0 {
		errs = append(errs, "юнит не привязан ни к одному зданию для найма")
	}

	return errs
}

func (s *UnitService) HasUnsavedChanges() bool {
	return s.repo.HasUnsavedChanges()
}

func (s *UnitService) Save() error {
	return s.repo.Save()
}
