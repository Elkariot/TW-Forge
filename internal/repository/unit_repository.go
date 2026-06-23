package repository

import (
	"fmt"
	"slices"
	"strings"
	"tw-forge/internal/domain"
)

func (r *InMemoryRepository) GetAllUnits() []domain.Unit {
	var units []domain.Unit
	for _, u := range r.working.Units {
		if !u.IsDeleted {
			units = append(units, u)
		}
	}
	return units
}

func (r *InMemoryRepository) GetDeletedUnits() []domain.Unit {
	var units []domain.Unit
	for _, u := range r.working.Units {
		if u.IsDeleted {
			units = append(units, u)
		}
	}
	return units
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

func (r *InMemoryRepository) DeleteUnit(unitType, faction string) error {
	for i, u := range r.working.Units {
		if u.Type != unitType {
			continue
		}

		// Remove faction from ownership
		newOwnership := make([]string, 0, len(u.Ownership))
		for _, f := range u.Ownership {
			if f != faction {
				newOwnership = append(newOwnership, f)
			}
		}

		// Remove unit from buildings for this specific faction
		for gi := range r.working.Buildings {
			for li := range r.working.Buildings[gi].Levels {
				lvl := &r.working.Buildings[gi].Levels[li]
				var keep []domain.RecruitSlot
				for _, s := range lvl.RecruitSlots {
					if s.UnitType == unitType && slices.Contains(s.Requirements, faction) {
						continue
					}
					keep = append(keep, s)
				}
				lvl.RecruitSlots = keep
			}
		}

		// Check if any non-slave factions remain
		hasRealFaction := false
		for _, f := range newOwnership {
			if strings.ToLower(f) != "slave" {
				hasRealFaction = true
				break
			}
		}

		if !hasRealFaction {
			// Full soft delete: clear ownership, mark deleted, purge from all remaining buildings
			r.working.Units[i].Ownership = nil
			r.working.Units[i].IsDeleted = true
			for gi := range r.working.Buildings {
				for li := range r.working.Buildings[gi].Levels {
					lvl := &r.working.Buildings[gi].Levels[li]
					var keep []domain.RecruitSlot
					for _, s := range lvl.RecruitSlots {
						if s.UnitType != unitType {
							keep = append(keep, s)
						}
					}
					lvl.RecruitSlots = keep
				}
			}
			if r.changes[unitType] == ChangeAdded {
				r.working.Units = slices.Delete(r.working.Units, i, i+1)
				delete(r.changes, unitType)
				r.working.UnitRecruitIndex = domain.BuildRecruitIndex(r.working.Buildings)
				r.buildingsDirty = true
				return nil
			}
			r.changes[unitType] = ChangeModified
		} else {
			r.working.Units[i].Ownership = newOwnership
			if r.changes[unitType] != ChangeAdded {
				r.changes[unitType] = ChangeModified
			}
		}

		r.working.UnitRecruitIndex = domain.BuildRecruitIndex(r.working.Buildings)
		r.buildingsDirty = true
		return nil
	}
	return fmt.Errorf("unit %q not found", unitType)
}

func (r *InMemoryRepository) HardDeleteUnit(unitType string) error {
	for i, u := range r.working.Units {
		if u.Type != unitType {
			continue
		}
		_ = u
		buildingsChanged := false
		for gi := range r.working.Buildings {
			for li := range r.working.Buildings[gi].Levels {
				lvl := &r.working.Buildings[gi].Levels[li]
				var keep []domain.RecruitSlot
				for _, s := range lvl.RecruitSlots {
					if s.UnitType != unitType {
						keep = append(keep, s)
					} else {
						buildingsChanged = true
					}
				}
				lvl.RecruitSlots = keep
			}
		}
		r.working.Units = slices.Delete(r.working.Units, i, i+1)
		if r.changes[unitType] == ChangeAdded {
			delete(r.changes, unitType)
		} else {
			r.changes[unitType] = ChangeDeleted
		}
		if buildingsChanged {
			r.working.UnitRecruitIndex = domain.BuildRecruitIndex(r.working.Buildings)
			r.buildingsDirty = true
		}
		return nil
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
		for i, u := range r.working.Units {
			if u.Type == unitType {
				r.working.Units = slices.Delete(r.working.Units, i, i+1)
				break
			}
		}
	case ChangeModified:
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
		r.restoreUnitBuildingSlots(unitType)
	case ChangeDeleted:
		for _, u := range r.original.Units {
			if u.Type == unitType {
				r.working.Units = append(r.working.Units, u)
				break
			}
		}
		r.restoreUnitBuildingSlots(unitType)
	}

	delete(r.changes, unitType)
	return nil
}

func (r *InMemoryRepository) restoreUnitBuildingSlots(unitType string) {
	type key struct{ group, level string }
	origSlots := make(map[key][]domain.RecruitSlot)
	for _, g := range r.original.Buildings {
		for _, l := range g.Levels {
			for _, s := range l.RecruitSlots {
				if s.UnitType == unitType {
					k := key{g.Name, l.Name}
					origSlots[k] = append(origSlots[k], s)
				}
			}
		}
	}
	for gi := range r.working.Buildings {
		for li := range r.working.Buildings[gi].Levels {
			lvl := &r.working.Buildings[gi].Levels[li]
			k := key{r.working.Buildings[gi].Name, lvl.Name}
			var keep []domain.RecruitSlot
			for _, s := range lvl.RecruitSlots {
				if s.UnitType != unitType {
					keep = append(keep, s)
				}
			}
			if orig, ok := origSlots[k]; ok {
				keep = append(keep, orig...)
			}
			lvl.RecruitSlots = keep
		}
	}
	r.working.UnitRecruitIndex = domain.BuildRecruitIndex(r.working.Buildings)
	r.buildingsDirty = true
}