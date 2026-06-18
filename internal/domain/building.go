package domain

type BuildingGroup struct {
	Name   string
	Levels []BuildingLevel
}

type BuildingLevel struct {
	Name             string
	RequiredCultures []string
	RecruitSlots     []RecruitSlot
	Construction     int
	Cost             int
	SettlementMin    string
	Upgrades         []string
}

// Requirements содержит культуры и/или названия фракций — игра принимает оба варианта в одном синтаксисе.
type RecruitSlot struct {
	UnitType     string
	Level        int
	Requirements []string
	Conditions   string
}

type RecruitLocation struct {
	GroupName string
	LevelName string
}

func BuildRecruitIndex(buildings []BuildingGroup) map[string][]RecruitLocation {
	idx := make(map[string][]RecruitLocation)
	for _, g := range buildings {
		for _, l := range g.Levels {
			for _, s := range l.RecruitSlots {
				idx[s.UnitType] = append(idx[s.UnitType], RecruitLocation{
					GroupName: g.Name,
					LevelName: l.Name,
				})
			}
		}
	}
	return idx
}

// CultureBuildingIndex — культура → названия BuildingGroup, у которых есть уровень с этой культурой.
type CultureBuildingIndex map[string][]string

func BuildCultureBuildingIndex(buildings []BuildingGroup) CultureBuildingIndex {
	idx := make(CultureBuildingIndex)
	for _, g := range buildings {
		added := make(map[string]bool)
		for _, l := range g.Levels {
			for _, culture := range l.RequiredCultures {
				if !added[culture] {
					idx[culture] = append(idx[culture], g.Name)
					added[culture] = true
				}
			}
		}
	}
	return idx
}
