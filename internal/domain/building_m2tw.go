package domain

type M2TWBuildingGroup struct {
	Name        string
	DisplayName string
	ConvertTo   string
	Levels      []M2TWBuildingLevel
}

type M2TWBuildingLevel struct {
	Name             string
	DisplayName      string
	SettlementType   string // "city" or "castle"
	RequiredCultures []string
	Dependency       *BuildingDependency
	DependencyInline bool // true when dependency appears on the level header line, not inside the block
	ConvertTo        int
	RecruitPools     []M2TWRecruitPool
	BonusLines       []string
	Construction     int
	Cost             int
	SettlementMin    string
	Upgrades         []string
}

// M2TWRecruitPool maps to: recruit_pool "UnitType" InitPool ReplenishRate MaxPool Exp [requires ...]
type M2TWRecruitPool struct {
	UnitType      string
	InitialPool   int
	ReplenishRate float64
	MaxPool       int
	ExpGained     int
	Factions      []string
	Conditions    string // e.g. "and region_religion catholic 30" or "and hidden_resource xbritain"
}

func BuildM2TWRecruitIndex(buildings []M2TWBuildingGroup) map[string][]RecruitLocation {
	idx := make(map[string][]RecruitLocation)
	for _, g := range buildings {
		for _, l := range g.Levels {
			for _, p := range l.RecruitPools {
				idx[p.UnitType] = append(idx[p.UnitType], RecruitLocation{
					GroupName: g.Name,
					LevelName: l.Name,
				})
			}
		}
	}
	return idx
}

func copyM2TWBuilding(b M2TWBuildingGroup) M2TWBuildingGroup {
	levels := make([]M2TWBuildingLevel, len(b.Levels))
	for i, l := range b.Levels {
		l.RequiredCultures = copyStrings(l.RequiredCultures)
		l.Upgrades = copyStrings(l.Upgrades)
		l.BonusLines = copyStrings(l.BonusLines)
		pools := make([]M2TWRecruitPool, len(l.RecruitPools))
		for j, p := range l.RecruitPools {
			p.Factions = copyStrings(p.Factions)
			pools[j] = p
		}
		l.RecruitPools = pools
		levels[i] = l
	}
	b.Levels = levels
	return b
}
