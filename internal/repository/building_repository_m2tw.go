package repository

import (
	"fmt"
	"tw-forge/internal/domain"
)

func (r *InMemoryM2TWRepository) GetBuildings() []domain.M2TWBuildingGroup {
	return r.working.Buildings
}

func (r *InMemoryM2TWRepository) GetBuildingByName(name string) (domain.M2TWBuildingGroup, bool) {
	for _, b := range r.working.Buildings {
		if b.Name == name {
			return b, true
		}
	}
	return domain.M2TWBuildingGroup{}, false
}

func (r *InMemoryM2TWRepository) GetUnitBuildings(unitType string) []domain.RecruitLocation {
	if locs := r.working.UnitRecruitIndex[unitType]; locs != nil {
		return locs
	}
	return []domain.RecruitLocation{}
}

func (r *InMemoryM2TWRepository) GetFactionBuildings(faction string) []domain.M2TWBuildingGroup {
	var result []domain.M2TWBuildingGroup
	for _, g := range r.working.Buildings {
		for _, lvl := range g.Levels {
			if len(lvl.RequiredFactions) == 0 {
				result = append(result, g)
				goto nextGroup
			}
			for _, f := range lvl.RequiredFactions {
				if f == faction {
					result = append(result, g)
					goto nextGroup
				}
			}
		}
	nextGroup:
	}
	return result
}

func (r *InMemoryM2TWRepository) UpdateBuildingLevel(groupName, levelName string, pools []domain.M2TWRecruitPool, bonusLines []string) error {
	for i := range r.working.Buildings {
		if r.working.Buildings[i].Name != groupName {
			continue
		}
		for j := range r.working.Buildings[i].Levels {
			if r.working.Buildings[i].Levels[j].Name != levelName {
				continue
			}
			r.working.Buildings[i].Levels[j].RecruitPools = pools
			r.working.Buildings[i].Levels[j].BonusLines = bonusLines
			r.working.UnitRecruitIndex = domain.BuildM2TWRecruitIndex(r.working.Buildings)
			r.buildingsDirty = true
			return nil
		}
		return fmt.Errorf("level %q not found in group %q", levelName, groupName)
	}
	return fmt.Errorf("building group %q not found", groupName)
}

func (r *InMemoryM2TWRepository) UpdateBuildingLevelProps(groupName, levelName string, cost, construction, convertTo int, settlementMin, settlementType string, requiredFactions []string, dependencyGroup, dependencyLevel string, upgrades []string) error {
	for i := range r.working.Buildings {
		if r.working.Buildings[i].Name != groupName {
			continue
		}
		for j := range r.working.Buildings[i].Levels {
			if r.working.Buildings[i].Levels[j].Name != levelName {
				continue
			}
			lvl := &r.working.Buildings[i].Levels[j]
			lvl.Cost = cost
			lvl.Construction = construction
			lvl.ConvertTo = convertTo
			lvl.SettlementMin = settlementMin
			lvl.SettlementType = settlementType
			lvl.RequiredFactions = requiredFactions
			if dependencyGroup != "" {
				lvl.Dependency = &domain.BuildingDependency{Group: dependencyGroup, Level: dependencyLevel}
			} else {
				lvl.Dependency = nil
			}
			lvl.Upgrades = upgrades
			r.buildingsDirty = true
			return nil
		}
		return fmt.Errorf("level %q not found in group %q", levelName, groupName)
	}
	return fmt.Errorf("building group %q not found", groupName)
}

func (r *InMemoryM2TWRepository) RevertBuildings() error {
	orig := r.original.DeepCopy()
	r.working.Buildings = orig.Buildings
	r.working.UnitRecruitIndex = domain.BuildM2TWRecruitIndex(r.working.Buildings)
	r.buildingsDirty = false
	if r.writer == nil {
		return nil
	}
	return r.writer.SaveM2TWBuildingsDraft(r.working)
}
