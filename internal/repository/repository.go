package repository

import (
	"fmt"
	"modding-utils/internal/domain"
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
	GetUnitsByFaction(faction string) ([]domain.Unit, error)
	GetUnitByType(unitType string) (domain.Unit, bool)

	// Buildings
	GetBuildings() []domain.BuildingGroup
	GetBuildingByName(name string) (domain.BuildingGroup, bool)
	GetUnitBuildings(unitType string) []domain.RecruitLocation
	GetCultureBuildings(culture string) []domain.BuildingGroup

	// Mutations
	AddUnit(unit domain.Unit) error
	UpdateUnit(originalType string, unit domain.Unit) error
	DeleteUnit(unitType string) error

	// Revert
	RevertUnit(unitType string) error
	RevertAll()

	// Persistence
	HasUnsavedChanges() bool
	GetChanges() map[string]ChangeType
	SaveDraft() error
	Save() error
}

type Writer interface {
	SaveDraft(data domain.GameData, changes map[string]ChangeType) error
	Apply() error
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

func (r *InMemoryRepository) SaveDraft() error {
	if r.writer == nil {
		return fmt.Errorf("writer not configured")
	}
	return r.writer.SaveDraft(r.working, r.changes)
}

func (r *InMemoryRepository) Save() error {
	if r.writer == nil {
		return fmt.Errorf("writer not configured")
	}
	return r.writer.Apply()
}
