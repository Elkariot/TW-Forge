package repository

import (
	"tw-forge/internal/domain"
)

type M2TWGameRepository interface {
	// Reference data
	GetReligions() []string
	GetHiddenResources() []string
	GetCultureNames() []string
	GetProjectileTypes() []string

	// Factions
	GetFactions() []domain.M2TWFaction
	GetFactionByName(name string) (domain.M2TWFaction, bool)

	// Units
	GetAllUnits() []domain.M2TWUnit
	GetDeletedUnits() []domain.M2TWUnit
	GetUnitsByFaction(faction string) ([]domain.M2TWUnit, error)
	GetUnitByType(unitType string) (domain.M2TWUnit, bool)

	// Buildings
	GetBuildings() []domain.M2TWBuildingGroup
	GetBuildingByName(name string) (domain.M2TWBuildingGroup, bool)
	GetUnitBuildings(unitType string) []domain.RecruitLocation
	GetFactionBuildings(faction string) []domain.M2TWBuildingGroup
	UpdateBuildingLevel(groupName, levelName string, pools []domain.M2TWRecruitPool, bonusLines []string) error
	UpdateBuildingLevelProps(groupName, levelName string, cost, construction, convertTo int, settlementMin, settlementType string, requiredFactions []string, dependencyGroup, dependencyLevel string, upgrades []string) error
	RevertBuildings() error

	// Mutations
	AddUnit(unit domain.M2TWUnit) error
	UpdateUnit(originalType string, unit domain.M2TWUnit) error
	DeleteUnit(unitType, faction string) error
	HardDeleteUnit(unitType string) error

	// Revert
	RevertUnit(unitType string) error
	RevertAll()

	// State
	HasUnsavedChanges() bool
	GetChanges() map[string]ChangeType
	GetData() domain.M2TWGameData
	IsBuildingsDirty() bool
	CommitSave()
}

type InMemoryM2TWRepository struct {
	original       domain.M2TWGameData
	working        domain.M2TWGameData
	changes        map[string]ChangeType
	buildingsDirty bool
}

func NewM2TW(data domain.M2TWGameData) *InMemoryM2TWRepository {
	return &InMemoryM2TWRepository{
		original: data,
		working:  data.DeepCopy(),
		changes:  make(map[string]ChangeType),
	}
}

func (r *InMemoryM2TWRepository) RevertAll() {
	r.working = r.original.DeepCopy()
	r.changes = make(map[string]ChangeType)
	r.buildingsDirty = false
}

func (r *InMemoryM2TWRepository) HasUnsavedChanges() bool {
	return len(r.changes) > 0 || r.buildingsDirty
}

func (r *InMemoryM2TWRepository) GetChanges() map[string]ChangeType {
	return r.changes
}

func (r *InMemoryM2TWRepository) GetData() domain.M2TWGameData {
	return r.working
}

func (r *InMemoryM2TWRepository) IsBuildingsDirty() bool {
	return r.buildingsDirty
}

func (r *InMemoryM2TWRepository) CommitSave() {
	r.original = r.working.DeepCopy()
	r.changes = make(map[string]ChangeType)
	r.buildingsDirty = false
}
