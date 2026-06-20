package domain

// M2TWFaction includes the religion field absent in RTW.
type M2TWFaction struct {
	Name            string
	DisplayName     string
	Culture         string
	Religion        string
	PrimaryColour   Colour
	SecondaryColour Colour
}

type M2TWGameData struct {
	Units            []M2TWUnit
	Buildings        []M2TWBuildingGroup
	Factions         []M2TWFaction
	Religions        []string
	HiddenResources  []string
	UnitRecruitIndex map[string][]RecruitLocation
	Cultures         []string
	ProjectileTypes  []string
}

func (g M2TWGameData) DeepCopy() M2TWGameData {
	units := make([]M2TWUnit, len(g.Units))
	for i, u := range g.Units {
		units[i] = CopyM2TWUnit(u)
	}

	buildings := make([]M2TWBuildingGroup, len(g.Buildings))
	for i, b := range g.Buildings {
		buildings[i] = copyM2TWBuilding(b)
	}

	factions := make([]M2TWFaction, len(g.Factions))
	copy(factions, g.Factions)

	idx := make(map[string][]RecruitLocation, len(g.UnitRecruitIndex))
	for k, v := range g.UnitRecruitIndex {
		copied := make([]RecruitLocation, len(v))
		copy(copied, v)
		idx[k] = copied
	}

	return M2TWGameData{
		Units:            units,
		Buildings:        buildings,
		Factions:         factions,
		Religions:        copyStrings(g.Religions),
		HiddenResources:  copyStrings(g.HiddenResources),
		UnitRecruitIndex: idx,
		Cultures:         copyStrings(g.Cultures),
		ProjectileTypes:  copyStrings(g.ProjectileTypes),
	}
}
