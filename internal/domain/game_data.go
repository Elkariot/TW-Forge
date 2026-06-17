package domain

type GameData struct {
	Units     []Unit
	Buildings []BuildingGroup
	Factions  []Faction
}

func (g GameData) DeepCopy() GameData {
	units := make([]Unit, len(g.Units))
	for i, u := range g.Units {
		units[i] = copyUnit(u)
	}

	buildings := make([]BuildingGroup, len(g.Buildings))
	for i, b := range g.Buildings {
		buildings[i] = copyBuilding(b)
	}

	factions := make([]Faction, len(g.Factions))
	copy(factions, g.Factions)

	return GameData{
		Units:     units,
		Buildings: buildings,
		Factions:  factions,
	}
}

func copyUnit(u Unit) Unit {
	u.Officers = copyStrings(u.Officers)
	u.Attributes = copyStrings(u.Attributes)
	u.StatPriAttr = copyStrings(u.StatPriAttr)
	u.StatSecAttr = copyStrings(u.StatSecAttr)
	u.Ownership = copyStrings(u.Ownership)
	return u
}

func copyBuilding(b BuildingGroup) BuildingGroup {
	levels := make([]BuildingLevel, len(b.Levels))
	for i, l := range b.Levels {
		l.RequiredFactions = copyStrings(l.RequiredFactions)
		l.Upgrades = copyStrings(l.Upgrades)
		slots := make([]RecruitSlot, len(l.RecruitSlots))
		for j, s := range l.RecruitSlots {
			s.Cultures = copyStrings(s.Cultures)
			slots[j] = s
		}
		l.RecruitSlots = slots
		levels[i] = l
	}
	b.Levels = levels
	return b
}

func copyStrings(s []string) []string {
	if s == nil {
		return nil
	}
	c := make([]string, len(s))
	copy(c, s)
	return c
}
