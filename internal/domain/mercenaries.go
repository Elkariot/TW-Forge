package domain

// MercenaryUnit is one "unit <name> exp N [armour N] [weapon_lvl N] cost N replenish
// A - B max N initial N [end_year N] [start_year N] [religions { ... }] [crusading]
// [events { ... }]" line within a mercenary pool. RTW only ever uses the fields up to
// Initial; the rest are M2TW-only (EndYear..Events) or REX/M2EX-only (Armour, WeaponLvl
// — https://github.com/Pannoniae/rex) and stay nil/empty unless present in the file.
type MercenaryUnit struct {
	Name string

	Exp           int
	Cost          int
	ReplenishLow  float64
	ReplenishHigh float64
	Max           int
	Initial       int

	Armour    *int // REX/M2EX only
	WeaponLvl *int // REX/M2EX only

	EndYear   *int // M2TW only
	StartYear *int // M2TW only
	Religions []string
	Crusading bool
	Events    []string
}

// MercenaryPool is one "pool <Name> / regions ... / unit ... (repeated)" block.
// RawBlock holds the original text (CRLF-joined, no trailing blank lines) so unchanged
// pools are written back byte-for-byte; it's regenerated from the typed fields only when
// the pool is dirty (see writer.SaveMercenariesDraft).
type MercenaryPool struct {
	Name    string
	Regions []string
	Units   []MercenaryUnit

	IsDeleted bool
	RawBlock  string
}

// MercenaryFile is the parsed content of one campaign's descr_mercenaries.txt.
// Header is the leading comment block, preserved verbatim.
type MercenaryFile struct {
	Header string
	Pools  []MercenaryPool
}

func (f MercenaryFile) DeepCopy() MercenaryFile {
	pools := make([]MercenaryPool, len(f.Pools))
	for i, p := range f.Pools {
		pools[i] = copyMercenaryPool(p)
	}
	return MercenaryFile{Header: f.Header, Pools: pools}
}

func copyMercenaryPool(p MercenaryPool) MercenaryPool {
	p.Regions = copyStrings(p.Regions)
	units := make([]MercenaryUnit, len(p.Units))
	for i, u := range p.Units {
		units[i] = copyMercenaryUnit(u)
	}
	p.Units = units
	return p
}

func copyMercenaryUnit(u MercenaryUnit) MercenaryUnit {
	u.Religions = copyStrings(u.Religions)
	u.Events = copyStrings(u.Events)
	if u.Armour != nil {
		v := *u.Armour
		u.Armour = &v
	}
	if u.WeaponLvl != nil {
		v := *u.WeaponLvl
		u.WeaponLvl = &v
	}
	if u.EndYear != nil {
		v := *u.EndYear
		u.EndYear = &v
	}
	if u.StartYear != nil {
		v := *u.StartYear
		u.StartYear = &v
	}
	return u
}
