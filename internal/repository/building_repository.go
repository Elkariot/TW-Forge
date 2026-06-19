package repository

import (
	"fmt"
	"modding-utils/internal/domain"
)

func (r *InMemoryRepository) GetBuildings() []domain.BuildingGroup {
	return r.working.Buildings
}

func (r *InMemoryRepository) GetBuildingByName(name string) (domain.BuildingGroup, bool) {
	for _, b := range r.working.Buildings {
		if b.Name == name {
			return b, true
		}
	}
	return domain.BuildingGroup{}, false
}

func (r *InMemoryRepository) GetUnitBuildings(unitType string) []domain.RecruitLocation {
	if locs := r.working.UnitRecruitIndex[unitType]; locs != nil {
		return locs
	}
	return []domain.RecruitLocation{}
}

func (r *InMemoryRepository) GetCultureBuildings(culture string) []domain.BuildingGroup {
	names := r.working.CultureBuildingIndex[culture]
	result := make([]domain.BuildingGroup, 0, len(names))
	for _, name := range names {
		if b, ok := r.GetBuildingByName(name); ok {
			result = append(result, b)
		}
	}
	return result
}

func (r *InMemoryRepository) UpdateBuildingLevel(groupName, levelName string, slots []domain.RecruitSlot) error {
	for i := range r.working.Buildings {
		if r.working.Buildings[i].Name != groupName {
			continue
		}
		for j := range r.working.Buildings[i].Levels {
			if r.working.Buildings[i].Levels[j].Name != levelName {
				continue
			}
			r.working.Buildings[i].Levels[j].RecruitSlots = slots
			r.working.UnitRecruitIndex = domain.BuildRecruitIndex(r.working.Buildings)
			r.buildingsDirty = true
			return nil
		}
		return fmt.Errorf("level %q not found in group %q", levelName, groupName)
	}
	return fmt.Errorf("building group %q not found", groupName)
}

func (r *InMemoryRepository) RevertBuildings() error {
	orig := r.original.DeepCopy()
	r.working.Buildings = orig.Buildings
	r.working.UnitRecruitIndex = domain.BuildRecruitIndex(r.working.Buildings)
	r.working.CultureBuildingIndex = domain.BuildCultureBuildingIndex(r.working.Buildings)
	r.buildingsDirty = false
	if r.writer == nil {
		return nil
	}
	return r.writer.SaveBuildingsDraft(r.working)
}