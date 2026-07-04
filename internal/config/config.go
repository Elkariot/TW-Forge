package config

type GameVersion int
type Operations int

const (
	Medieval GameVersion = iota
	Rome
)

const (
	UpdateUnit Operations = iota
	CopyUnit
	CreateUnit
	DeleteUnit
	HardDeleteUnit
	UpdateBuildingLevel
)

var OperationType = map[Operations]string {
	UpdateUnit: "update_unit",
	CopyUnit: "copy_unit",
	CreateUnit: "create_unit",
	DeleteUnit: "delete_unit",
	HardDeleteUnit: "hard_delete_unit",
	UpdateBuildingLevel: "update_building_level",
}
