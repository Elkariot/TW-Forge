package repository

import "modding-utils/internal/domain"

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