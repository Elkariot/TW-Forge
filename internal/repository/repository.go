package repository

import (
	"fmt"
	"modding-utils/internal/domain"
	"slices"
)

type ChangeType int

const (
	ChangeModified ChangeType = iota
	ChangeAdded
	ChangeDeleted
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
	GetChanges() map[string]ChangeType
	Save() error
}

type Writer interface {
	Write(data domain.GameData, changes map[string]ChangeType) error
}

type InMemoryRepository struct {
	original domain.GameData
	working  domain.GameData
	writer   Writer
	changes  map[string]ChangeType
}

func New(data domain.GameData, writer Writer) *InMemoryRepository {
	return &InMemoryRepository{
		original: data,
		working:  data.DeepCopy(),
		writer:   writer,
		changes:  make(map[string]ChangeType),
	}
}

func (r *InMemoryRepository) GetFactions() []domain.Faction {
	return r.working.Factions
}

func (r *InMemoryRepository) GetFactionByName(name string) (domain.Faction, bool) {
	for _, f := range r.working.Factions {
		if f.Name == name {
			return f, true
		}
	}
	return domain.Faction{}, false
}

func (r *InMemoryRepository) GetAllUnits() []domain.Unit {
	return r.working.Units
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

func (r *InMemoryRepository) GetBuildings() []domain.BuildingGroup {
	panic("not implemented")
}

func (r *InMemoryRepository) GetBuildingByName(name string) (domain.BuildingGroup, bool) {
	panic("not implemented")
}

func (r *InMemoryRepository) AddUnit(unit domain.Unit) error {
	if _, exists := r.GetUnitByType(unit.Type); exists {
		return fmt.Errorf("unit with type %q already exists", unit.Type)
	}
	r.working.Units = append(r.working.Units, unit)
	r.changes[unit.Type] = ChangeAdded
	return nil
}

func (r *InMemoryRepository) UpdateUnit(unit domain.Unit) error {
	for i, u := range r.working.Units {
		if u.Type == unit.Type {
			r.working.Units[i] = unit
			// не перезаписываем ChangeAdded — новый юнит остаётся новым
			if r.changes[unit.Type] != ChangeAdded {
				r.changes[unit.Type] = ChangeModified
			}
			return nil
		}
	}
	return fmt.Errorf("unit %q not found", unit.Type)
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

func (r *InMemoryRepository) RevertAll() {
	r.working = r.original.DeepCopy()
	r.changes = make(map[string]ChangeType)
}

func (r *InMemoryRepository) HasUnsavedChanges() bool {
	return len(r.changes) > 0
}

func (r *InMemoryRepository) GetChanges() map[string]ChangeType {
	return r.changes
}

func (r *InMemoryRepository) Save() error {
	if r.writer == nil {
		return fmt.Errorf("writer not configured")
	}
	return r.writer.Write(r.working, r.changes)
}
