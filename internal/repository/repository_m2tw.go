package repository

import (
	"fmt"
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
	SaveBuildingsDraft() error

	// Mutations
	AddUnit(unit domain.M2TWUnit) error
	UpdateUnit(originalType string, unit domain.M2TWUnit) error
	DeleteUnit(unitType, faction string) error
	HardDeleteUnit(unitType string) error
	CopyBattleModel(srcType, dstType string) error
	DeleteBattleModel(unitType string) error
	PatchSoldierFaction(soldierModel, srcFaction, dstFaction string) error
	CopyUnitCard(srcUnitType, dstUnitType, srcFaction, dstFaction string) error
	RenameUnitCard(oldType, newType, faction string) error
	DeleteUnitAssets(unitType string) error

	// Revert
	RevertUnit(unitType string) error
	RevertAll()

	// Persistence
	HasUnsavedChanges() bool
	GetChanges() map[string]ChangeType
	SaveDraft() error
	Save() error
}

type M2TWWriter interface {
	SaveM2TWDraft(data domain.M2TWGameData, changes map[string]ChangeType) error
	SaveM2TWBuildingsDraft(data domain.M2TWGameData) error
	CopyBattleModelDraft(srcName, dstName string) error
	DeleteBattleModelDraft(unitType string) error
	PatchSoldierFactionDraft(soldierModel, srcFaction, dstFaction string) error
	CopyUnitCardDraft(srcUnitType, dstUnitType, srcFaction, dstFaction string) error
	RenameUnitCardDraft(oldType, newType, faction string) error
	DeleteUnitAssets(unitType string) error
	Apply() error
}

type InMemoryM2TWRepository struct {
	original       domain.M2TWGameData
	working        domain.M2TWGameData
	writer         M2TWWriter
	changes        map[string]ChangeType
	buildingsDirty bool
}

func NewM2TW(data domain.M2TWGameData, writer M2TWWriter) *InMemoryM2TWRepository {
	return &InMemoryM2TWRepository{
		original: data,
		working:  data.DeepCopy(),
		writer:   writer,
		changes:  make(map[string]ChangeType),
	}
}

func (r *InMemoryM2TWRepository) RevertAll() {
	r.working = r.original.DeepCopy()
	r.changes = make(map[string]ChangeType)
	r.buildingsDirty = false
	if r.writer != nil {
		_ = r.writer.SaveM2TWBuildingsDraft(r.working)
	}
}

func (r *InMemoryM2TWRepository) HasUnsavedChanges() bool {
	return len(r.changes) > 0 || r.buildingsDirty
}

func (r *InMemoryM2TWRepository) GetChanges() map[string]ChangeType {
	return r.changes
}

func (r *InMemoryM2TWRepository) SaveDraft() error {
	if r.writer == nil {
		return fmt.Errorf("writer not configured")
	}
	return r.writer.SaveM2TWDraft(r.working, r.changes)
}

func (r *InMemoryM2TWRepository) SaveBuildingsDraft() error {
	if !r.buildingsDirty {
		return nil
	}
	if r.writer == nil {
		return fmt.Errorf("writer not configured")
	}
	return r.writer.SaveM2TWBuildingsDraft(r.working)
}

func (r *InMemoryM2TWRepository) CopyBattleModel(srcType, dstType string) error {
	if r.writer == nil {
		return fmt.Errorf("writer not configured")
	}
	return r.writer.CopyBattleModelDraft(srcType, dstType)
}

func (r *InMemoryM2TWRepository) PatchSoldierFaction(soldierModel, srcFaction, dstFaction string) error {
	if r.writer == nil {
		return nil
	}
	return r.writer.PatchSoldierFactionDraft(soldierModel, srcFaction, dstFaction)
}

func (r *InMemoryM2TWRepository) DeleteBattleModel(unitType string) error {
	if r.writer == nil {
		return nil
	}
	return r.writer.DeleteBattleModelDraft(unitType)
}

func (r *InMemoryM2TWRepository) CopyUnitCard(srcUnitType, dstUnitType, srcFaction, dstFaction string) error {
	if r.writer == nil {
		return nil
	}
	return r.writer.CopyUnitCardDraft(srcUnitType, dstUnitType, srcFaction, dstFaction)
}

func (r *InMemoryM2TWRepository) RenameUnitCard(oldType, newType, faction string) error {
	if r.writer == nil {
		return nil
	}
	return r.writer.RenameUnitCardDraft(oldType, newType, faction)
}

func (r *InMemoryM2TWRepository) DeleteUnitAssets(unitType string) error {
	if r.writer == nil {
		return nil
	}
	return r.writer.DeleteUnitAssets(unitType)
}

func (r *InMemoryM2TWRepository) Save() error {
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
