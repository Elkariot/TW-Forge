package repository

import (
	"fmt"
	"modding-utils/internal/domain"
	"slices"
)

type GameRepository interface {
	// Factions
	GetFactions() []domain.Faction
	GetFactionByName(name string) (domain.Faction, bool)

	// Units
	GetAllUnits() []domain.Unit
	GetUnitsByFaction(faction string) ([]domain.Unit, error)
	GetUnitByType(unitType string) (domain.Unit, bool)

	// Buildings
	GetBuildings() []domain.BuildingGroup
	GetBuildingByName(name string) (domain.BuildingGroup, bool)

	// Mutations
	AddUnit(unit domain.Unit) error
	UpdateUnit(unit domain.Unit) error
	DeleteUnit(unitType string) error

	// Revert
	RevertUnit(unitType string) error
	RevertAll()

	// Persistence
	HasUnsavedChanges() bool
	Save() error
}

type InMemoryRepository struct {
	original domain.GameData
	working  domain.GameData
	writer   Writer
}

type Writer interface {
	Write(data domain.GameData) error
}

func New(data domain.GameData, writer Writer) *InMemoryRepository {
	return &InMemoryRepository{
		original: data,
		working:  data.DeepCopy(),
		writer:   writer,
	}
}

func (r *InMemoryRepository) GetFactions() []domain.Faction {
	return r.working.Factions
}

func (r *InMemoryRepository) GetFactionByName(name string) (domain.Faction, bool) {
	var faction domain.Faction
	var isFind bool

	for _, f := range r.working.Factions {
		if f.Name == name {
			faction = f
			isFind = true
			return faction, isFind
		}
	}

	return faction, isFind
}

func (r *InMemoryRepository) GetAllUnits() []domain.Unit {
	return r.working.Units
}

func (r *InMemoryRepository) GetUnitsByFaction(faction string) ([]domain.Unit, error) {
	if _, isFind := r.GetFactionByName(faction); !isFind {
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

func (r *InMemoryRepository) GetBuildings() []domain.BuildingGroup {
	panic("not implemented")
}

func (r *InMemoryRepository) GetBuildingByName(name string) (domain.BuildingGroup, bool) {
	panic("not implemented")
}

func (r *InMemoryRepository) AddUnit(unit domain.Unit) error {
	panic("not implemented")
}

func (r *InMemoryRepository) UpdateUnit(unit domain.Unit) error {
	panic("not implemented")
}

func (r *InMemoryRepository) DeleteUnit(unitType string) error {
	panic("not implemented")
}

func (r *InMemoryRepository) RevertUnit(unitType string) error {
	panic("not implemented")
}

func (r *InMemoryRepository) RevertAll() {
	panic("not implemented")
}

func (r *InMemoryRepository) HasUnsavedChanges() bool {
	panic("not implemented")
}

func (r *InMemoryRepository) Save() error {
	panic("not implemented")
}
