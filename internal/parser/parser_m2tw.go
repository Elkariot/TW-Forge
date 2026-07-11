package parser

import (
	"bufio"
	"fmt"
	"tw-forge/internal/domain"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ParseM2TW parses all game data files for Medieval 2 Total War.
func (p *Parser) ParseM2TW() (*domain.M2TWGameData, error) {
	factions, err := p.parseM2TWFactions()
	if err != nil {
		return nil, fmt.Errorf("parse factions: %w", err)
	}

	textNames := p.parseFactionDisplayNames()
	for i, f := range factions {
		if name, ok := textNames[strings.ToUpper(f.Name)]; ok {
			factions[i].DisplayName = name
		}
	}

	units, err := p.parseM2TWUnits()
	if err != nil {
		return nil, fmt.Errorf("parse units: %w", err)
	}

	unitTexts, _ := p.parseM2TWUnitTexts()
	for i, u := range units {
		if t, ok := unitTexts[u.Dictionary]; ok {
			units[i].Name = t.Name
			units[i].Descr = t.Descr
			units[i].DescrShort = t.DescrShort
		}
	}

	buildings, hiddenResources, err := p.parseM2TWBuildings()
	if err != nil {
		return nil, fmt.Errorf("parse buildings: %w", err)
	}

	religions := p.parseReligions()

	for i := range buildings {
		if name, ok := textNames[strings.ToUpper(buildings[i].Name)]; ok {
			buildings[i].DisplayName = name
		}
		for j := range buildings[i].Levels {
			if name, ok := textNames[strings.ToUpper(buildings[i].Levels[j].Name)]; ok {
				buildings[i].Levels[j].DisplayName = name
			}
		}
	}

	return &domain.M2TWGameData{
		Units:            units,
		Buildings:        buildings,
		Factions:         factions,
		Religions:        religions,
		HiddenResources:  hiddenResources,
		UnitRecruitIndex: domain.BuildM2TWRecruitIndex(buildings),
		Cultures:         p.parseCultureNames(),
		ProjectileTypes:  p.parseProjectileTypes(),
	}, nil
}

// ── Factions ─────────────────────────────────────────────────────────────────

func (p *Parser) parseM2TWFactions() ([]domain.M2TWFaction, error) {
	path := filepath.Join(p.GamePath, "descr_sm_factions.txt")
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var factions []domain.M2TWFaction
	var cur domain.M2TWFaction
	inBlock := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			if inBlock && cur.Name != "" {
				factions = append(factions, cur)
				cur = domain.M2TWFaction{}
				inBlock = false
			}
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "faction":
			if inBlock && cur.Name != "" {
				factions = append(factions, cur)
				cur = domain.M2TWFaction{}
			}
			cur.Name = fields[1]
			inBlock = true
		case "culture":
			cur.Culture = fields[1]
		case "religion":
			cur.Religion = fields[1]
		case "primary_colour":
			cur.PrimaryColour = parseColour(fields[1:])
		case "secondary_colour":
			cur.SecondaryColour = parseColour(fields[1:])
		}
	}
	if inBlock && cur.Name != "" {
		factions = append(factions, cur)
	}
	return factions, scanner.Err()
}

// ── Religions ─────────────────────────────────────────────────────────────────

func (p *Parser) parseReligions() []string {
	path := filepath.Join(p.GamePath, "descr_religions.txt")
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var religions []string
	inBlock := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		fields := strings.Fields(line)
		if fields[0] == "religions" {
			inBlock = true
			continue
		}
		if inBlock {
			if line == "{" {
				continue
			}
			if line == "}" {
				inBlock = false
				continue
			}
			if len(fields) == 1 {
				religions = append(religions, fields[0])
			}
		}
		// Also handle "religion <name>" blocks that define individual religions
	}
	return religions
}

// ── Units ─────────────────────────────────────────────────────────────────────

func (p *Parser) parseM2TWUnits() ([]domain.M2TWUnit, error) {
	path := filepath.Join(p.GamePath, "export_descr_unit.txt")
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var units []domain.M2TWUnit
	var unit domain.M2TWUnit
	inUnit := false

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if i := strings.Index(line, ";"); i != -1 {
			line = strings.TrimSpace(line[:i])
			if line == "" {
				continue
			}
		}

		fields := strings.Fields(line)
		keyword := fields[0]

		switch keyword {
		case "type":
			// Финализируем предыдущий юнит перед началом нового.
			if inUnit && unit.Type != "" {
				units = append(units, unit)
			}
			unit = domain.M2TWUnit{Eras: make(map[string][]string)}
			unit.Type = strings.Join(fields[1:], " ")
			inUnit = true

		case "ownership":
			if !inUnit {
				continue
			}
			for _, f := range fields[1:] {
				faction := trimComma(f)
				if faction != "" {
					unit.Ownership = append(unit.Ownership, faction)
				}
			}
			// Не финализируем здесь — era/info_pic_dir/recruit_priority_offset идут после ownership.

		default:
			if !inUnit {
				continue
			}
			switch keyword {
			case "dictionary":
				unit.Dictionary = strings.Join(fields[1:], " ")
			case "category":
				unit.Category = fields[1]
			case "class":
				unit.Class = fields[1]
			case "voice_type":
				if len(fields) > 1 {
					unit.VoiceType = fields[1]
				}
			case "accent":
				if len(fields) > 1 {
					unit.Accent = fields[1]
				}
			case "banner":
				if len(fields) >= 3 {
					switch fields[1] {
					case "faction":
						unit.BannerFaction = fields[2]
					case "holy":
						unit.BannerHoly = fields[2]
					}
				}
			case "move_speed_mod":
				if len(fields) > 1 {
					unit.MoveSpeedMod, _ = strconv.ParseFloat(trimComma(fields[1]), 64)
				}
			case "officer":
				unit.Officers = append(unit.Officers, strings.Join(fields[1:], " "))
			case "soldier":
				vals := splitByComma(fields)
				if len(vals) >= 4 {
					unit.Soldier.Model = vals[0]
					unit.Soldier.Count = parseInt(vals[1])
					unit.Soldier.Extras = parseInt(vals[2])
					unit.Soldier.Mass = parseFloat(vals[3])
				}
			case "mount":
				unit.Mount = strings.Join(fields[1:], " ")
			case "mount_effect":
				unit.MountEffect = strings.Join(fields[1:], " ")
			case "attributes":
				for _, v := range splitByComma(fields) {
					unit.Attributes = append(unit.Attributes, v)
				}
			case "formation":
				unit.Formation = strings.Join(fields[1:], " ")
			case "stat_health":
				vals := splitByComma(fields)
				if len(vals) >= 2 {
					unit.StatHealth = [2]int{parseInt(vals[0]), parseInt(vals[1])}
				}
			case "stat_pri":
				unit.StatPri = parseWeaponStats(splitByComma(fields))
			case "stat_pri_attr":
				for _, f := range fields[1:] {
					unit.StatPriAttr = append(unit.StatPriAttr, trimComma(f))
				}
			case "stat_sec":
				unit.StatSec = parseWeaponStats(splitByComma(fields))
			case "stat_sec_attr":
				for _, f := range fields[1:] {
					unit.StatSecAttr = append(unit.StatSecAttr, trimComma(f))
				}
			case "stat_pri_armour":
				if len(fields) >= 5 {
					unit.StatPriArmour = domain.ArmourStats{
						Armour:   parseInt(fields[1]),
						DefSkill: parseInt(fields[2]),
						Shield:   parseInt(fields[3]),
						Sound:    trimComma(fields[4]),
					}
				}
			case "stat_sec_armour":
				if len(fields) >= 4 {
					unit.StatSecArmour = domain.ArmourStats{
						Armour:   parseInt(fields[1]),
						DefSkill: parseInt(fields[2]),
						Sound:    trimComma(fields[3]),
					}
				}
			case "stat_heat":
				unit.StatHeat = parseInt(fields[1])
			case "stat_ground":
				vals := splitByComma(fields)
				if len(vals) >= 4 {
					unit.StatGround = [4]int{parseInt(vals[0]), parseInt(vals[1]), parseInt(vals[2]), parseInt(vals[3])}
				}
			case "stat_mental":
				vals := splitByComma(fields)
				if len(vals) >= 3 {
					unit.StatMental = domain.MentalStats{
						Morale:     parseInt(vals[0]),
						Discipline: vals[1],
						Training:   vals[2],
					}
				}
			case "stat_charge_dist":
				unit.StatChargeDist = parseInt(fields[1])
			case "stat_fire_delay":
				unit.StatFireDelay = parseInt(fields[1])
			case "stat_food":
				vals := splitByComma(fields)
				if len(vals) >= 2 {
					unit.StatFood = [2]int{parseInt(vals[0]), parseInt(vals[1])}
				}
			case "stat_cost":
				vals := splitByComma(fields)
				c := &unit.StatCost
				if len(vals) > 0 {
					c.Turns = parseInt(vals[0])
				}
				if len(vals) > 1 {
					c.Cost = parseInt(vals[1])
				}
				if len(vals) > 2 {
					c.Upkeep = parseInt(vals[2])
				}
				if len(vals) > 3 {
					c.WeaponUpgrade = parseInt(vals[3])
				}
				if len(vals) > 4 {
					c.ArmourUpgrade = parseInt(vals[4])
				}
				if len(vals) > 5 {
					c.Custom = parseInt(vals[5])
				}
				if len(vals) > 6 {
					c.Extra1 = parseInt(vals[6])
				}
				if len(vals) > 7 {
					c.Extra2 = parseInt(vals[7])
				}
			case "armour_ug_levels":
				unit.ArmourUgLevels = strings.Join(fields[1:], " ")
			case "armour_ug_models":
				unit.ArmourUgModels = strings.Join(fields[1:], " ")
			case "era":
				// "era 0   england, scotland, france, ..."
				if len(fields) < 3 {
					continue
				}
				eraKey := strings.TrimSpace(fields[1])
				if unit.Eras == nil {
					unit.Eras = make(map[string][]string)
				}
				for _, f := range fields[2:] {
					faction := trimComma(f)
					if faction != "" {
						unit.Eras[eraKey] = append(unit.Eras[eraKey], faction)
					}
				}
			case "info_pic_dir":
				if len(fields) > 1 {
					unit.InfoPicDir = fields[1]
				}
			case "recruit_priority_offset":
				if len(fields) > 1 {
					unit.RecruitPriorityOffset = parseInt(fields[1])
				}
			}
		}
	}
	// Финализируем последний юнит в файле.
	if inUnit && unit.Type != "" {
		units = append(units, unit)
	}
	return units, scanner.Err()
}

// ── Unit texts ────────────────────────────────────────────────────────────────

// parseM2TWUnitTexts reads the M2TW unit description text file.
// M2TW stores unit text in data/text/export_units.txt (UTF-16LE).
func (p *Parser) parseM2TWUnitTexts() (map[string]unitText, error) {
	return p.parseUnitTexts()
}

// ── Buildings ────────────────────────────────────────────────────────────────

func (p *Parser) parseM2TWBuildings() ([]domain.M2TWBuildingGroup, []string, error) {
	path := filepath.Join(p.GamePath, "export_descr_buildings.txt")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil // no EDB in this mod/game — buildings editor just shows empty
		}
		return nil, nil, err
	}
	defer file.Close()

	var buildings []domain.M2TWBuildingGroup
	var cur domain.M2TWBuildingGroup
	var curLevel domain.M2TWBuildingLevel
	var hiddenResources []string

	depth := 0
	inCapability := false
	inUpgrades := false
	var pendingLevelName string
	var pendingLevelFactions []string
	var pendingLevelType string // "city" or "castle"
	var pendingConvertTo int
	var pendingDependency *domain.BuildingDependency

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)

		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if i := strings.Index(line, ";"); i != -1 {
			line = strings.TrimSpace(line[:i])
			if line == "" {
				continue
			}
		}

		switch line {
		case "{":
			depth++
			if depth == 3 && pendingLevelName != "" {
				curLevel = domain.M2TWBuildingLevel{
					Name:             pendingLevelName,
					SettlementType:   pendingLevelType,
					RequiredFactions: pendingLevelFactions,
					Dependency:       pendingDependency,
					DependencyInline: pendingDependency != nil,
					ConvertTo:        pendingConvertTo,
				}
				pendingLevelName = ""
				pendingLevelFactions = nil
				pendingLevelType = ""
				pendingConvertTo = 0
				pendingDependency = nil
			}
			continue
		case "}":
			if depth == 4 {
				inCapability = false
				inUpgrades = false
			} else if depth == 3 {
				cur.Levels = append(cur.Levels, curLevel)
			} else if depth == 1 {
				buildings = append(buildings, cur)
			}
			depth--
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch depth {
		case 0:
			if fields[0] == "hidden_resources" {
				for _, r := range fields[1:] {
					hiddenResources = append(hiddenResources, r)
				}
			} else if fields[0] == "building" && len(fields) > 1 {
				cur = domain.M2TWBuildingGroup{Name: fields[1]}
			}

		case 1:
			if fields[0] == "convert_to" && len(fields) > 1 {
				cur.ConvertTo = fields[1]
			}

		case 2:
			// "level_name city|castle requires factions { ... }"
			// OR just "levels wooden_wall ..."
			if fields[0] == "levels" {
				continue
			}
			pendingLevelName = fields[0]
			// Check for city/castle keyword after level name
			for _, f := range fields[1:] {
				switch f {
				case "city":
					pendingLevelType = "city"
				case "castle":
					pendingLevelType = "castle"
				}
			}
			pendingLevelFactions = extractFactions(line)
			pendingDependency = extractBuildingDependency(line)
			pendingConvertTo = 0
			if idx := strings.Index(line, "convert_to"); idx != -1 {
				parts := strings.Fields(line[idx:])
				if len(parts) >= 2 {
					pendingConvertTo, _ = strconv.Atoi(parts[1])
				}
			}

		case 3:
			switch fields[0] {
			case "capability":
				inCapability = true
			case "upgrades":
				inUpgrades = true
			case "convert_to":
				if len(fields) > 1 {
					curLevel.ConvertTo, _ = strconv.Atoi(fields[1])
				}
			case "construction":
				curLevel.Construction = parseInt(fields[1])
			case "cost":
				curLevel.Cost = parseInt(fields[1])
			case "settlement_min":
				curLevel.SettlementMin = fields[1]
			case "building_present_min_level":
				if len(fields) >= 3 {
					curLevel.Dependency = &domain.BuildingDependency{Group: fields[1], Level: fields[2]}
				}
			}

		case 4:
			if inCapability {
				if fields[0] == "recruit_pool" {
					if pool, ok := parseM2TWRecruitPool(line); ok {
						curLevel.RecruitPools = append(curLevel.RecruitPools, pool)
					}
				} else {
					curLevel.BonusLines = append(curLevel.BonusLines, line)
				}
			} else if inUpgrades {
				curLevel.Upgrades = append(curLevel.Upgrades, trimComma(fields[0]))
			}
		}
	}

	return buildings, hiddenResources, scanner.Err()
}

// parseM2TWRecruitPool parses a recruit_pool line.
// Format: recruit_pool "Unit Name" init_pool replenish_rate max_pool exp requires factions { ... } [conditions]
func parseM2TWRecruitPool(line string) (domain.M2TWRecruitPool, bool) {
	// Extract quoted unit name
	start := strings.Index(line, `"`)
	end := strings.LastIndex(line, `"`)
	if start == -1 || start == end {
		return domain.M2TWRecruitPool{}, false
	}
	unitType := line[start+1 : end]
	rest := strings.TrimSpace(line[end+1:])

	// Parse numeric fields: init_pool replenish_rate max_pool exp
	parts := strings.Fields(rest)
	var pool domain.M2TWRecruitPool
	pool.UnitType = unitType

	idx := 0
	if idx < len(parts) {
		pool.InitialPool, _ = strconv.Atoi(parts[idx])
		idx++
	}
	if idx < len(parts) {
		pool.ReplenishRate, _ = strconv.ParseFloat(parts[idx], 64)
		idx++
	}
	if idx < len(parts) {
		pool.MaxPool, _ = strconv.Atoi(parts[idx])
		idx++
	}
	if idx < len(parts) {
		pool.ExpGained, _ = strconv.Atoi(parts[idx])
		idx++
	}

	// Find "requires factions { ... }" part
	reqIdx := strings.Index(rest, "requires")
	if reqIdx == -1 {
		return pool, true
	}
	reqPart := strings.TrimSpace(rest[reqIdx+len("requires"):])

	// Extract factions list from { ... }
	factStart := strings.Index(reqPart, "{")
	factEnd := strings.Index(reqPart, "}")
	if factStart != -1 && factEnd != -1 && factEnd > factStart {
		for _, f := range strings.Split(reqPart[factStart+1:factEnd], ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				pool.Factions = append(pool.Factions, f)
			}
		}
		pool.Conditions = strings.TrimSpace(reqPart[factEnd+1:])
	}

	return pool, true
}
