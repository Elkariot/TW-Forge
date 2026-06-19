package repository

import (
	"fmt"
	"modding-utils/internal/domain"
	"slices"
)

func (r *InMemoryRepository) GetAllUnits() []domain.Unit {
	return r.working.Units
}

func (r *InMemoryRepository) GetAllUnitsTypes() []string {
	units := r.GetAllUnits()
	unitsTypes := make([]string, len(units))

	for _, unit := range units {
		unitsTypes = append(unitsTypes, unit.Type)
	}

	return unitsTypes
}

func (r *InMemoryRepository) GetUnitsByFaction(faction string) ([]domain.Unit, error) {
	if _, ok := r.GetFactionByName(faction); !ok {
		return nil, fmt.Errorf("faction not found")
	}

	var units []domain.Unit
	for _, unit := range r.working.Units {
		if slices.Contains(unit.Ownership, faction) {
			units = append(units, unit)
		}
	}
	return units, nil
}

func (r *InMemoryRepository) GetUnitByType(unitType string) (domain.Unit, bool) {
	for _, u := range r.working.Units {
		if u.Type == unitType {
			return u, true
		}
	}
	return domain.Unit{}, false
}

func (r *InMemoryRepository) AddUnit(unit domain.Unit) error {
	if _, exists := r.GetUnitByType(unit.Type); exists {
		return fmt.Errorf("unit with type %q already exists", unit.Type)
	}
	r.working.Units = append(r.working.Units, unit)
	r.changes[unit.Type] = ChangeAdded
	return nil
}

func (r *InMemoryRepository) UpdateUnit(originalType string, unit domain.Unit) error {
	for i, u := range r.working.Units {
		if u.Type == originalType {
			r.working.Units[i] = unit
			if originalType != unit.Type {
				// Переименование: удаляем старый блок из файла, добавляем новый в конец
				r.changes[originalType] = ChangeDeleted
				r.changes[unit.Type] = ChangeAdded
				// Обновляем индекс: переносим локации под новый ключ
				if locs, ok := r.working.UnitRecruitIndex[originalType]; ok {
					r.working.UnitRecruitIndex[unit.Type] = locs
					delete(r.working.UnitRecruitIndex, originalType)
				}
			} else if r.changes[originalType] != ChangeAdded {
				r.changes[originalType] = ChangeModified
			}
			return nil
		}
	}
	return fmt.Errorf("unit %q not found", originalType)
}

func (r *InMemoryRepository) DeleteUnit(unitType string) error {
	for i, u := range r.working.Units {
		if u.Type == unitType {
			r.working.Units = slices.Delete(r.working.Units, i, i+1)
			r.changes[unitType] = ChangeDeleted
			return nil
		}
	}
	return fmt.Errorf("unit %q not found", unitType)
}

func (r *InMemoryRepository) RevertUnit(unitType string) error {
	changeType, isChanged := r.changes[unitType]
	if !isChanged {
		return nil
	}

	switch changeType {
	case ChangeAdded:
		// просто удаляем из working, в original его не было
		for i, u := range r.working.Units {
			if u.Type == unitType {
				r.working.Units = slices.Delete(r.working.Units, i, i+1)
				break
			}
		}
	case ChangeModified:
		// восстанавливаем из original
		for _, u := range r.original.Units {
			if u.Type == unitType {
				for i, wu := range r.working.Units {
					if wu.Type == unitType {
						r.working.Units[i] = u
						break
					}
				}
				break
			}
		}
	case ChangeDeleted:
		// возвращаем оригинал в конец списка
		for _, u := range r.original.Units {
			if u.Type == unitType {
				r.working.Units = append(r.working.Units, u)
				break
			}
		}
	}

	delete(r.changes, unitType)
	return nil
}