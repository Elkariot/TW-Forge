package service

import (
	"fmt"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"
	"strings"
)

type M2TWUnitService struct {
	repo repository.M2TWGameRepository
}

func NewM2TWUnitService(repo repository.M2TWGameRepository) *M2TWUnitService {
	return &M2TWUnitService{repo: repo}
}

func (s *M2TWUnitService) GetAllUnits() []domain.M2TWUnit {
	return s.repo.GetAllUnits()
}

func (s *M2TWUnitService) GetDeletedUnits() []domain.M2TWUnit {
	return s.repo.GetDeletedUnits()
}

func (s *M2TWUnitService) GetUnitsByFaction(faction string) ([]domain.M2TWUnit, error) {
	return s.repo.GetUnitsByFaction(faction)
}

func (s *M2TWUnitService) GetFactionUnitsByCategoryAndClass(faction string) (map[string]map[string][]domain.M2TWUnit, error) {
	units, err := s.repo.GetUnitsByFaction(faction)
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string][]domain.M2TWUnit)
	for _, unit := range units {
		if _, ok := result[unit.Category]; !ok {
			result[unit.Category] = make(map[string][]domain.M2TWUnit)
		}
		result[unit.Category][unit.Class] = append(result[unit.Category][unit.Class], unit)
	}
	return result, nil
}

func (s *M2TWUnitService) GetUnitByType(unitType string) (*domain.M2TWUnit, error) {
	unit, ok := s.repo.GetUnitByType(unitType)
	if !ok {
		return nil, fmt.Errorf("unit not found")
	}
	return &unit, nil
}

func (s *M2TWUnitService) Update(originalType string, unit domain.M2TWUnit) error {
	return s.repo.UpdateUnit(originalType, unit)
}

// TypeRenamed reports whether the type name changed — app.go uses this to decide whether to rename the icon.
func (s *M2TWUnitService) TypeRenamed(originalType string, unit domain.M2TWUnit) bool {
	return originalType != unit.Type
}

func (s *M2TWUnitService) CopyUnit(unitType, faction string) (newType string, srcFaction string, soldierModel string, err error) {
	unit, ok := s.repo.GetUnitByType(unitType)
	if !ok {
		return "", "", "", fmt.Errorf("unit not found: %s", unitType)
	}

	base := fmt.Sprintf("%s copy", unit.Type)
	newType = base
	for i := 2; ; i++ {
		if _, exists := s.repo.GetUnitByType(newType); !exists {
			break
		}
		newType = fmt.Sprintf("%s %d", base, i)
	}

	if len(unit.Ownership) > 0 {
		srcFaction = unit.Ownership[0]
	}
	soldierModel = unit.Soldier.Model

	unit.Type = newType
	unit.Dictionary = strings.ReplaceAll(newType, " ", "_")
	unit.Ownership = []string{faction}
	if len(unit.Eras) > 0 {
		unit.Eras = map[string][]string{"0": {faction}, "1": {faction}, "2": {faction}}
	}

	if err = s.repo.AddUnit(unit); err != nil {
		return "", "", "", fmt.Errorf("error copying unit: %w", err)
	}
	return newType, srcFaction, soldierModel, nil
}

func (s *M2TWUnitService) CreateUnit(templateType, newType, faction string) (srcFaction string, soldierModel string, err error) {
	if _, exists := s.repo.GetUnitByType(newType); exists {
		return "", "", fmt.Errorf("unit with type %q already exists", newType)
	}
	unit, ok := s.repo.GetUnitByType(templateType)
	if !ok {
		return "", "", fmt.Errorf("template %q not found", templateType)
	}
	if len(unit.Ownership) > 0 {
		srcFaction = unit.Ownership[0]
	}
	soldierModel = unit.Soldier.Model
	unit.Type = newType
	unit.Dictionary = strings.ReplaceAll(newType, " ", "_")
	unit.Ownership = []string{faction}
	if len(unit.Eras) > 0 {
		unit.Eras = map[string][]string{"0": {faction}, "1": {faction}, "2": {faction}}
	}
	if err = s.repo.AddUnit(unit); err != nil {
		return "", "", fmt.Errorf("failed to create unit: %w", err)
	}
	return srcFaction, soldierModel, nil
}

func (s *M2TWUnitService) Delete(unitType, faction string) error {
	if _, exists := s.repo.GetUnitByType(unitType); !exists {
		return fmt.Errorf("unit type not found: %s", unitType)
	}
	return s.repo.DeleteUnit(unitType, faction)
}

func (s *M2TWUnitService) HardDelete(unitType string) error {
	if _, exists := s.repo.GetUnitByType(unitType); !exists {
		return fmt.Errorf("unit type not found: %s", unitType)
	}
	return s.repo.HardDeleteUnit(unitType)
}

func (s *M2TWUnitService) Revert(unitType string) error {
	return s.repo.RevertUnit(unitType)
}

func (s *M2TWUnitService) RevertAll() {
	s.repo.RevertAll()
}

func (s *M2TWUnitService) GetUnitChangeType(unitType string) string {
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

func (s *M2TWUnitService) HasUnsavedChanges() bool {
	return s.repo.HasUnsavedChanges()
}

func (s *M2TWUnitService) Validate(unit domain.M2TWUnit, originalType string) []string {
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
