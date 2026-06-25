package service

import (
	"fmt"
	"strings"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"
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
	return s.repo.UpdateUnit(originalType, unit)
}

func (s *UnitService) Create(unit domain.Unit) error {
	if _, exists := s.repo.GetUnitByType(unit.Type); exists {
		return fmt.Errorf("unit type already in use: %s", unit.Type)
	}
	return s.repo.AddUnit(unit)
}

func (s *UnitService) Delete(unitType, faction string) error {
	if _, exists := s.repo.GetUnitByType(unitType); !exists {
		return fmt.Errorf("unit type not found: %s", unitType)
	}
	return s.repo.DeleteUnit(unitType, faction)
}

func (s *UnitService) HardDelete(unitType string) error {
	if _, exists := s.repo.GetUnitByType(unitType); !exists {
		return fmt.Errorf("unit type not found: %s", unitType)
	}
	return s.repo.HardDeleteUnit(unitType)
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

	return newUnitType, nil
}

// SoldierModel returns the soldier model of a unit — needed by app.go to pass to writer.
func (s *UnitService) SoldierModel(unitType string) string {
	if u, ok := s.repo.GetUnitByType(unitType); ok {
		return u.Soldier.Model
	}
	return ""
}

// FirstFaction returns the first faction of a unit — needed by app.go for icon rename.
func (s *UnitService) FirstFaction(unitType string) string {
	if u, ok := s.repo.GetUnitByType(unitType); ok && len(u.Ownership) > 0 {
		return u.Ownership[0]
	}
	return ""
}

func (s *UnitService) CreateUnit(templateType, newType, faction string) error {
	if _, exists := s.repo.GetUnitByType(newType); exists {
		return fmt.Errorf("unit with type %q already exists", newType)
	}
	unit, ok := s.repo.GetUnitByType(templateType)
	if !ok {
		return fmt.Errorf("template %q not found", templateType)
	}

	unit.Type = newType
	unit.Dictionary = strings.ReplaceAll(newType, " ", "_")
	unit.Ownership = []string{faction}
	return s.repo.AddUnit(unit)
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

func (s *UnitService) Validate(unit domain.Unit, originalType string) []string {
	var errs []string

	trimmedType := strings.TrimSpace(unit.Type)
	if trimmedType == "" {
		errs = append(errs, "unit type (internal id) is required")
	} else if trimmedType != originalType {
		if _, exists := s.repo.GetUnitByType(trimmedType); exists {
			errs = append(errs, fmt.Sprintf("type %q is already used by another unit", trimmedType))
		}
	}

	if strings.TrimSpace(unit.Name) == "" {
		errs = append(errs, "unit display name is required")
	}

	if len(unit.Ownership) == 0 {
		errs = append(errs, "unit must belong to at least one faction")
	} else {
		for _, faction := range unit.Ownership {
			if _, ok := s.repo.GetFactionByName(faction); !ok {
				errs = append(errs, fmt.Sprintf("faction %q not found", faction))
			}
		}
	}

	if len(s.repo.GetUnitBuildings(unit.Type)) == 0 {
		errs = append(errs, "unit is not assigned to any recruitment building")
	}

	return errs
}

func (s *UnitService) HasUnsavedChanges() bool {
	return s.repo.HasUnsavedChanges()
}
