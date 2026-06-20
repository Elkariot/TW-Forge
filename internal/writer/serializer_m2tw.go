package writer

import (
	"fmt"
	"tw-forge/internal/domain"
	"sort"
	"strings"
)

func serializeM2TWUnit(u domain.M2TWUnit) []string {
	var lines []string

	add := func(key, value string) {
		lines = append(lines, fmt.Sprintf("%-17s%s", key, value))
	}

	add("type", u.Type)
	add("dictionary", u.Dictionary)
	add("category", u.Category)
	add("class", u.Class)
	add("voice_type", u.VoiceType)
	if u.BannerFaction != "" {
		add("banner faction", u.BannerFaction)
	}
	if u.BannerHoly != "" {
		add("banner holy", u.BannerHoly)
	}
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
	if u.MoveSpeedMod != 0 {
		add("move_speed_mod", fmt.Sprintf("%.2f", u.MoveSpeedMod))
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
	add("stat_cost", fmt.Sprintf("%d, %d, %d, %d, %d, %d, %d, %d",
		u.StatCost.Turns, u.StatCost.Cost, u.StatCost.Upkeep,
		u.StatCost.WeaponUpgrade, u.StatCost.ArmourUpgrade, u.StatCost.Custom,
		u.StatCost.Extra1, u.StatCost.Extra2))
	if u.ArmourUgLevels != "" {
		add("armour_ug_levels", u.ArmourUgLevels)
	}
	if u.ArmourUgModels != "" {
		add("armour_ug_models", u.ArmourUgModels)
	}
	add("ownership", strings.Join(u.Ownership, ", "))

	// Era lines in sorted order (0, 1, 2)
	if len(u.Eras) > 0 {
		eraKeys := make([]int, 0, len(u.Eras))
		for k := range u.Eras {
			eraKeys = append(eraKeys, k)
		}
		sort.Ints(eraKeys)
		for _, era := range eraKeys {
			factions := u.Eras[era]
			if len(factions) > 0 {
				add(fmt.Sprintf("era %d", era), strings.Join(factions, ", "))
			}
		}
	}

	if u.RecruitPriorityOffset != 0 {
		add("recruit_priority_offset", fmt.Sprintf("%d", u.RecruitPriorityOffset))
	}

	return lines
}
