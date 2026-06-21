package writer

import (
	"fmt"
	"strconv"
	"strings"
	"tw-forge/internal/domain"
)

func serializeUnit(u domain.Unit) []string {
	var lines []string

	add := func(key, value string) {
		lines = append(lines, fmt.Sprintf("%-17s%s", key, value))
	}

	add("type", u.Type)
	if u.DictionaryComment != "" {
		add("dictionary", u.Dictionary+"      ; "+u.DictionaryComment)
	} else {
		add("dictionary", u.Dictionary)
	}
	add("category", u.Category)
	add("class", u.Class)
	add("voice_type", u.VoiceType)
	add("soldier", fmt.Sprintf("%s, %d, %d, %.1f",
		u.Soldier.Model, u.Soldier.Count, u.Soldier.Extras, u.Soldier.Mass))

	for _, o := range u.Officers {
		add("officer", o)
	}
	if u.Mount != "" {
		add("mount", u.Mount)
	}
	if u.MountEffect != "" {
		add("mount_effect", u.MountEffect)
	}

	if len(u.Attributes) > 0 {
		add("attributes", strings.Join(u.Attributes, ", "))
	}

	add("formation", u.Formation)
	add("stat_health", fmt.Sprintf("%d, %d", u.StatHealth[0], u.StatHealth[1]))
	add("stat_pri", serializeWeapon(u.StatPri))
	add("stat_pri_attr", joinOrNo(u.StatPriAttr))
	add("stat_sec", serializeWeapon(u.StatSec))
	add("stat_sec_attr", joinOrNo(u.StatSecAttr))
	add("stat_pri_armour", fmt.Sprintf("%d, %d, %d, %s",
		u.StatPriArmour.Armour, u.StatPriArmour.DefSkill, u.StatPriArmour.Shield, u.StatPriArmour.Sound))
	add("stat_sec_armour", fmt.Sprintf("%d, %d, %s",
		u.StatSecArmour.Armour, u.StatSecArmour.DefSkill, u.StatSecArmour.Sound))
	add("stat_heat", fmt.Sprintf("%d", u.StatHeat))
	add("stat_ground", fmt.Sprintf("%d, %d, %d, %d",
		u.StatGround[0], u.StatGround[1], u.StatGround[2], u.StatGround[3]))
	add("stat_mental", fmt.Sprintf("%d, %s, %s",
		u.StatMental.Morale, u.StatMental.Discipline, u.StatMental.Training))
	add("stat_charge_dist", fmt.Sprintf("%d", u.StatChargeDist))
	add("stat_fire_delay", fmt.Sprintf("%d", u.StatFireDelay))
	add("stat_food", fmt.Sprintf("%d, %d", u.StatFood[0], u.StatFood[1]))
	add("stat_cost", fmt.Sprintf("%d, %d, %d, %d, %d, %d",
		u.StatCost.Turns, u.StatCost.Cost, u.StatCost.Upkeep,
		u.StatCost.WeaponUpgrade, u.StatCost.ArmourUpgrade, u.StatCost.Custom))
	add("ownership", strings.Join(u.Ownership, ", "))

	return lines
}

func serializeWeapon(s domain.WeaponStats) string {
	return fmt.Sprintf("%d, %d, %s, %d, %d, %s, %s, %s, %s, %s, %s",
		s.Attack, s.ChargeBonus, s.Missile, s.Range, s.Ammo,
		s.WeaponType, s.TechType, s.DamageType, s.SoundType,
		rtwFloat(s.MinDelay), rtwFloat(s.Factor))
}

// rtwFloat форматирует число без лишних нулей: 50.0 → "50", 0.73 → "0.73".
func rtwFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func joinOrNo(s []string) string {
	if len(s) == 0 {
		return "no"
	}
	return strings.Join(s, ", ")
}


