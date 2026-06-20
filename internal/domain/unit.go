package domain

type Unit struct {
	Type       string
	Dictionary string

	Category  string
	Class     string
	VoiceType string

	Soldier     SoldierDef
	Officers    []string
	Mount       string
	MountEffect string

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
	StatCost      CostStats

	Ownership []string
	IsDeleted bool

	Name       string
	Descr      string
	DescrShort string
}

type SoldierDef struct {
	Model  string
	Count  int
	Extras int
	Mass   float64
}

type WeaponStats struct {
	Attack      int
	ChargeBonus int
	Missile     string
	Range       int
	Ammo        int
	WeaponType  string
	TechType    string
	DamageType  string
	SoundType   string
	MinDelay    float64
	Factor      float64
}

type ArmourStats struct {
	Armour   int
	DefSkill int
	Shield   int
	Sound    string
}

type MentalStats struct {
	Morale     int
	Discipline string
	Training   string
}

type CostStats struct {
	Turns         int
	Cost          int
	Upkeep        int
	WeaponUpgrade int
	ArmourUpgrade int
	Custom        int
}
