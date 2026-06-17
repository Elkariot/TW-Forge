package domain

type BuildingGroup struct {
	Name   string
	Levels []BuildingLevel
}

type BuildingLevel struct {
	Name             string
	RequiredFactions []string
	RecruitSlots     []RecruitSlot
	Construction     int
	Cost             int
	SettlementMin    string
	Upgrades         []string
}

type RecruitSlot struct {
	UnitType   string
	Level      int
	Factions   []string
	Conditions string
}
