package domain

type M2TWUnit struct {
	Type       string
	Dictionary string

	Category      string
	Class         string
	VoiceType     string
	Accent        string
	BannerFaction string
	BannerHoly    string

	Soldier      SoldierDef
	Officers     []string
	Mount        string
	MountEffect  string
	MoveSpeedMod float64

	Attributes []string
	Formation  string

	StatHealth    [2]int
	StatPri       WeaponStats
	StatPriAttr   []string
	StatSec       WeaponStats
	StatSecAttr   []string
	StatPriArmour ArmourStats
	StatSecArmour ArmourStats
	StatHeat      int
	StatGround    [4]int
	StatMental    MentalStats
	StatChargeDist int
	StatFireDelay  int
	StatFood      [2]int
	StatCost      M2TWCostStats

	ArmourUgLevels string // raw: "3, 9"
	ArmourUgModels string // raw: "NE_Bodyguard, NE_Bodyguard_ug1"

	Ownership             []string
	Eras                  map[string][]string // "0"/"1"/"2" → factions
	InfoPicDir            string
	RecruitPriorityOffset int

	IsDeleted  bool
	Name       string
	Descr      string
	DescrShort string
}

type M2TWCostStats struct {
	Turns         int
	Cost          int
	Upkeep        int
	WeaponUpgrade int
	ArmourUpgrade int
	Custom        int
	Extra1        int
	Extra2        int
}

func CopyM2TWUnit(u M2TWUnit) M2TWUnit {
	u.Officers = copyStrings(u.Officers)
	u.Attributes = copyStrings(u.Attributes)
	u.StatPriAttr = copyStrings(u.StatPriAttr)
	u.StatSecAttr = copyStrings(u.StatSecAttr)
	u.Ownership = copyStrings(u.Ownership)
	if u.Eras != nil {
		cp := make(map[string][]string, len(u.Eras))
		for k, v := range u.Eras {
			cp[k] = copyStrings(v)
		}
		u.Eras = cp
	}
	return u
}
