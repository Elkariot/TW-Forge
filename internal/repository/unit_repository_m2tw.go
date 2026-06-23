package repository

import (
	"fmt"
	"slices"
	"strings"
	"tw-forge/internal/domain"
)

func (r *InMemoryM2TWRepository) GetAllUnits() []domain.M2TWUnit {
	var units []domain.M2TWUnit
	for _, u := range r.working.Units {
		if !u.IsDeleted {
			units = append(units, u)
		}
	}
	return units
}

func (r *InMemoryM2TWRepository) GetDeletedUnits() []domain.M2TWUnit {
	var units []domain.M2TWUnit
	for _, u := range r.working.Units {
		if u.IsDeleted {
			units = append(units, u)
		}
	}
	return units
}

func (r *InMemoryM2TWRepository) GetUnitsByFaction(faction string) ([]domain.M2TWUnit, error) {
	if _, ok := r.GetFactionByName(faction); !ok {
		return nil, fmt.Errorf("faction not found")
	}
	var units []domain.M2TWUnit
	for _, u := range r.working.Units {
		if slices.Contains(u.Ownership, faction) {
			units = append(units, u)
		}
	}
	return units, nil
}

func (r *InMemoryM2TWRepository) GetUnitByType(unitType string) (domain.M2TWUnit, bool) {
	for _, u := range r.working.Units {
		if u.Type == unitType {
			return u, true
		}
	}
	return domain.M2TWUnit{}, false
}

func (r *InMemoryM2TWRepository) AddUnit(unit domain.M2TWUnit) error {
	if _, exists := r.GetUnitByType(unit.Type); exists {
		return fmt.Errorf("unit with type %q already exists", unit.Type)
	}
	r.working.Units = append(r.working.Units, unit)
	r.changes[unit.Type] = ChangeAdded
	return nil
}

func (r *InMemoryM2TWRepository) UpdateUnit(originalType string, unit domain.M2TWUnit) error {
	for i, u := range r.working.Units {
		if u.Type == originalType {
			r.working.Units[i] = unit
			if originalType != unit.Type {
				r.changes[originalType] = ChangeDeleted
				r.changes[unit.Type] = ChangeAdded
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

func (r *InMemoryM2TWRepository) DeleteUnit(unitType, faction string) error {
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
				var keep []domain.M2TWRecruitPool
				for _, p := range lvl.RecruitPools {
					if p.UnitType != unitType || !slices.Contains(p.Factions, faction) {
						keep = append(keep, p)
						continue
					}
					newFactions := slices.DeleteFunc(slices.Clone(p.Factions), func(f string) bool {
						return f == faction
					})
					if len(newFactions) == 0 {
						continue // drop pool entirely
					}
					p.Factions = newFactions
					keep = append(keep, p)
				}
				lvl.RecruitPools = keep
			}
		}

		// Remove faction from Eras map
		newEras := make(map[string][]string)
		for era, factions := range u.Eras {
			var ef []string
			for _, f := range factions {
				if f != faction {
					ef = append(ef, f)
				}
			}
			if len(ef) > 0 {
				newEras[era] = ef
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
			r.working.Units[i].Ownership = nil
			r.working.Units[i].Eras = nil
			r.working.Units[i].IsDeleted = true
			// Purge from all remaining buildings
			for gi := range r.working.Buildings {
				for li := range r.working.Buildings[gi].Levels {
					lvl := &r.working.Buildings[gi].Levels[li]
					var keep []domain.M2TWRecruitPool
					for _, p := range lvl.RecruitPools {
						if p.UnitType != unitType {
							keep = append(keep, p)
						}
					}
					lvl.RecruitPools = keep
				}
			}
			if r.changes[unitType] == ChangeAdded {
				r.working.Units = slices.Delete(r.working.Units, i, i+1)
				delete(r.changes, unitType)
				r.working.UnitRecruitIndex = domain.BuildM2TWRecruitIndex(r.working.Buildings)
				r.buildingsDirty = true
				return nil
			}
			r.changes[unitType] = ChangeModified
		} else {
			r.working.Units[i].Ownership = newOwnership
			r.working.Units[i].Eras = newEras
			if r.changes[unitType] != ChangeAdded {
				r.changes[unitType] = ChangeModified
			}
		}

		r.working.UnitRecruitIndex = domain.BuildM2TWRecruitIndex(r.working.Buildings)
		r.buildingsDirty = true
		return nil
	}
	return fmt.Errorf("unit %q not found", unitType)
}

func (r *InMemoryM2TWRepository) HardDeleteUnit(unitType string) error {
	for i, u := range r.working.Units {
		if u.Type != unitType {
			continue
		}
		_ = u
		// Remove from all buildings
		buildingsChanged := false
		for gi := range r.working.Buildings {
			for li := range r.working.Buildings[gi].Levels {
				lvl := &r.working.Buildings[gi].Levels[li]
				var keep []domain.M2TWRecruitPool
				for _, p := range lvl.RecruitPools {
					if p.UnitType != unitType {
						keep = append(keep, p)
					} else {
						buildingsChanged = true
					}
				}
				lvl.RecruitPools = keep
			}
		}
		r.working.Units = slices.Delete(r.working.Units, i, i+1)
		if r.changes[unitType] == ChangeAdded {
			delete(r.changes, unitType)
		} else {
			r.changes[unitType] = ChangeDeleted
		}
		if buildingsChanged {
			r.working.UnitRecruitIndex = domain.BuildM2TWRecruitIndex(r.working.Buildings)
			r.buildingsDirty = true
		}
		return nil
	}
	return fmt.Errorf("unit %q not found", unitType)
}

func (r *InMemoryM2TWRepository) RevertUnit(unitType string) error {
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
		r.restoreUnitBuildingPools(unitType)
	case ChangeDeleted:
		for _, u := range r.original.Units {
			if u.Type == unitType {
				r.working.Units = append(r.working.Units, u)
				break
			}
		}
		r.restoreUnitBuildingPools(unitType)
	}

	delete(r.changes, unitType)
	return nil
}

func (r *InMemoryM2TWRepository) restoreUnitBuildingPools(unitType string) {
	type key struct{ group, level string }
	origPools := make(map[key][]domain.M2TWRecruitPool)
	for _, g := range r.original.Buildings {
		for _, l := range g.Levels {
			for _, p := range l.RecruitPools {
				if p.UnitType == unitType {
					k := key{g.Name, l.Name}
					origPools[k] = append(origPools[k], p)
				}
			}
		}
	}
	for gi := range r.working.Buildings {
		for li := range r.working.Buildings[gi].Levels {
			lvl := &r.working.Buildings[gi].Levels[li]
			k := key{r.working.Buildings[gi].Name, lvl.Name}
			var keep []domain.M2TWRecruitPool
			for _, p := range lvl.RecruitPools {
				if p.UnitType != unitType {
					keep = append(keep, p)
				}
			}
			if orig, ok := origPools[k]; ok {
				keep = append(keep, orig...)
			}
			lvl.RecruitPools = keep
		}
	}
	r.working.UnitRecruitIndex = domain.BuildM2TWRecruitIndex(r.working.Buildings)
	r.buildingsDirty = true
}
