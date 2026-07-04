package repository

import (
	"tw-forge/internal/domain"
)

type ChangeType int

const (
	ChangeModified ChangeType = iota
	ChangeAdded
	ChangeDeleted
)

type GameRepository interface {
	// Reference data
	GetCultureNames() []string
	GetProjectileTypes() []string

	// Factions
	GetFactions() []domain.Faction
	GetFactionByName(name string) (domain.Faction, bool)
	GetFactionCulture(name string) (string, bool)

	// Units
	GetAllUnits() []domain.Unit
	GetDeletedUnits() []domain.Unit
	GetUnitsByFaction(faction string) ([]domain.Unit, error)
	GetUnitByType(unitType string) (domain.Unit, bool)

	// Buildings
	GetBuildings() []domain.BuildingGroup
	GetBuildingByName(name string) (domain.BuildingGroup, bool)
	GetUnitBuildings(unitType string) []domain.RecruitLocation
	GetCultureBuildings(culture string) []domain.BuildingGroup
	UpdateBuildingLevel(groupName, levelName string, slots []domain.RecruitSlot) error
	UpdateBuildingLevelProps(groupName, levelName string, cost, construction int, settlementMin string, requiredCultures []string, dependencyGroup, dependencyLevel string, upgrades, bonusLines []string) error
	RevertBuildings() error

	// Mutations
	AddUnit(unit domain.Unit) error
	UpdateUnit(originalType string, unit domain.Unit) error
	DeleteUnit(unitType, faction string) error
	HardDeleteUnit(unitType string) error

	// Revert
	RevertUnit(unitType string) error
	RevertAll()

	// State
	HasUnsavedChanges() bool
	GetChanges() map[string]ChangeType
	GetData() domain.GameData
	IsBuildingsDirty() bool
	CommitSave()
}

type InMemoryRepository struct {
	original       domain.GameData
	working        domain.GameData
	changes        map[string]ChangeType
	buildingsDirty bool
}

func New(data domain.GameData) *InMemoryRepository {
	return &InMemoryRepository{
		original: data,
		working:  data.DeepCopy(),
		changes:  make(map[string]ChangeType),
	}
}

func (r *InMemoryRepository) RevertAll() {
	r.working = r.original.DeepCopy()
	r.changes = make(map[string]ChangeType)
	r.buildingsDirty = false
}

func (r *InMemoryRepository) HasUnsavedChanges() bool {
	return len(r.changes) > 0 || r.buildingsDirty
}

func (r *InMemoryRepository) GetChanges() map[string]ChangeType {
	return r.changes
}

func (r *InMemoryRepository) GetData() domain.GameData {
	return r.working
}

func (r *InMemoryRepository) IsBuildingsDirty() bool {
	return r.buildingsDirty
}

func (r *InMemoryRepository) CommitSave() {
	r.original = r.working.DeepCopy()
	r.changes = make(map[string]ChangeType)
	r.buildingsDirty = false
}
