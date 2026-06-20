package service

import (
	"fmt"
	"modding-utils/internal/domain"
	"modding-utils/internal/repository"
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
	if err := s.repo.UpdateUnit(originalType, unit); err != nil {
		return err
	}
	return s.repo.SaveDraft()
}

func (s *M2TWUnitService) CopyUnit(unitType, faction string) (string, error) {
	unit, ok := s.repo.GetUnitByType(unitType)
	if !ok {
		return "", fmt.Errorf("unit not found: %s", unitType)
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
	unit.Eras = map[int][]string{0: {faction}, 1: {faction}, 2: {faction}}

	if err := s.repo.AddUnit(unit); err != nil {
		return "", fmt.Errorf("error copying unit: %w", err)
	}

	// Copy battle model entry — best-effort, не все юниты имеют запись в modeldb.
	_ = s.repo.CopyBattleModel(unitType, newUnitType)

	return newUnitType, s.repo.SaveDraft()
}

func (s *M2TWUnitService) CreateUnit(templateType, newType, faction string) error {
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
	unit.Eras = map[int][]string{0: {faction}, 1: {faction}, 2: {faction}}
	if err := s.repo.AddUnit(unit); err != nil {
		return fmt.Errorf("ошибка создания юнита: %w", err)
	}
	return s.repo.SaveDraft()
}

func (s *M2TWUnitService) Delete(unitType string) error {
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

func (s *M2TWUnitService) Save() error {
	return s.repo.Save()
}

func (s *M2TWUnitService) Validate(unit domain.M2TWUnit, originalType string) []string {
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
