package repository

import (
	"fmt"
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
	GetAllUnitsTypes() []string
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
	SaveBuildingsDraft() error

	// Mutations
	AddUnit(unit domain.Unit) error
	UpdateUnit(originalType string, unit domain.Unit) error
	DeleteUnit(unitType, faction string) error
	HardDeleteUnit(unitType string) error
	DeleteUnitAssets(unitType string) error

	// Revert
	RevertUnit(unitType string) error
	RevertAll()

	// Persistence
	HasUnsavedChanges() bool
	GetChanges() map[string]ChangeType
	SaveDraft() error
	Save() error

	// Model assets (RTW)
	CopyUnitModelAssets(soldierModel, srcUnitType, dstUnitType, srcFaction, dstFaction string) error
	RenameUnitIcon(oldType, newType, faction string) error
}

type Writer interface {
	SaveDraft(data domain.GameData, changes map[string]ChangeType) error
	SaveBuildingsDraft(data domain.GameData) error
	Apply() error
	CopyUnitModelAssets(soldierModel, srcUnitType, dstUnitType, srcFaction, dstFaction string) error
	RenameUnitIcon(oldType, newType, faction string) error
	DeleteUnitAssets(unitType string) error
}

type InMemoryRepository struct {
	original       domain.GameData
	working        domain.GameData
	writer         Writer
	changes        map[string]ChangeType
	buildingsDirty bool
}

func New(data domain.GameData, writer Writer) *InMemoryRepository {
	return &InMemoryRepository{
		original: data,
		working:  data.DeepCopy(),
		writer:   writer,
		changes:  make(map[string]ChangeType),
	}
}

func (r *InMemoryRepository) RevertAll() {
	r.working = r.original.DeepCopy()
	r.changes = make(map[string]ChangeType)
	r.buildingsDirty = false
	if r.writer != nil {
		_ = r.writer.SaveBuildingsDraft(r.working)
	}
}

func (r *InMemoryRepository) HasUnsavedChanges() bool {
	return len(r.changes) > 0 || r.buildingsDirty
}

func (r *InMemoryRepository) GetChanges() map[string]ChangeType {
	return r.changes
}

func (r *InMemoryRepository) SaveDraft() error {
	if r.writer == nil {
		return fmt.Errorf("writer not configured")
	}
	return r.writer.SaveDraft(r.working, r.changes)
}

func (r *InMemoryRepository) SaveBuildingsDraft() error {
	if r.writer == nil {
		return fmt.Errorf("writer not configured")
	}
	return r.writer.SaveBuildingsDraft(r.working)
}

func (r *InMemoryRepository) Save() error {
	if r.writer == nil {
		return fmt.Errorf("writer not configured")
	}
	if err := r.writer.Apply(); err != nil {
		return err
	}
	// Reset state: applied changes are now the new baseline.
	r.original = r.working.DeepCopy()
	r.changes = make(map[string]ChangeType)
	r.buildingsDirty = false
	return nil
}

func (r *InMemoryRepository) CopyUnitModelAssets(soldierModel, srcUnitType, dstUnitType, srcFaction, dstFaction string) error {
	if r.writer == nil {
		return nil
	}
	return r.writer.CopyUnitModelAssets(soldierModel, srcUnitType, dstUnitType, srcFaction, dstFaction)
}

func (r *InMemoryRepository) RenameUnitIcon(oldType, newType, faction string) error {
	if r.writer == nil {
		return nil
	}
	return r.writer.RenameUnitIcon(oldType, newType, faction)
}

func (r *InMemoryRepository) DeleteUnitAssets(unitType string) error {
	if r.writer == nil {
		return nil
	}
	return r.writer.DeleteUnitAssets(unitType)
}
