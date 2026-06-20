package parser

import (
	"bufio"
	"fmt"
	"tw-forge/internal/config"
	"tw-forge/internal/domain"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type unitText struct {
	Name       string
	Descr      string
	DescrShort string
}

type Parser struct {
	GameVersion config.GameVersion
	GamePath    string
}

func New(gameVersion config.GameVersion, gamePath string) *Parser {
	return &Parser{
		GameVersion: gameVersion,
		GamePath:    gamePath,
	}
}

func (p *Parser) ParseTextFiles() (*domain.GameData, error) {
	factions, err := p.parseFactions()
	if err != nil {
		return nil, fmt.Errorf("parse factions error: %w", err)
	}

	textNames := p.parseFactionDisplayNames()
	for i, f := range factions {
		if name, ok := textNames[strings.ToUpper(f.Name)]; ok {
			factions[i].DisplayName = name
		}
	}

	units, err := p.parseUnits()
	if err != nil {
		return nil, fmt.Errorf("parse units error: %w", err)
	}

	buildings, err := p.parseBuildings()
	if err != nil {
		return nil, fmt.Errorf("parse buildings error: %w", err)
	}

	// Enrich building names from text files (keys are uppercase level/group names)
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

	return &domain.GameData{
		Units:                units,
		Buildings:            buildings,
		Factions:             factions,
		UnitRecruitIndex:     domain.BuildRecruitIndex(buildings),
		CultureBuildingIndex: domain.BuildCultureBuildingIndex(buildings),
		Cultures:             p.parseCultureNames(),
		ProjectileTypes:      p.parseProjectileTypes(),
	}, nil
}

var defaultCultures = []string{"roman", "barbarian", "carthaginian", "greek", "egyptian", "eastern"}

func (p *Parser) parseCultureNames() []string {
	path := filepath.Join(p.GamePath, "descr_cultures.txt")
	file, err := os.Open(path)
	if err != nil {
		return defaultCultures
	}
	defer file.Close()

	var cultures []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "culture" {
			cultures = append(cultures, fields[1])
		}
	}
	if len(cultures) == 0 {
		return defaultCultures
	}
	return cultures
}

var defaultProjectiles = []string{"no", "arrow", "javelin", "pilum", "bullet", "stone", "head", "bolt", "dart", "ballista", "scorpion"}

func (p *Parser) parseProjectileTypes() []string {
	path := filepath.Join(p.GamePath, "descr_projectile_new.txt")
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join(p.GamePath, "descr_projectile.txt")
	}
	file, err := os.Open(path)
	if err != nil {
		return defaultProjectiles
	}
	defer file.Close()

	seen := map[string]bool{"no": true}
	projectiles := []string{"no"}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "projectile" {
			name := fields[1]
			if !seen[name] {
				seen[name] = true
				projectiles = append(projectiles, name)
			}
		}
	}
	if len(projectiles) <= 1 {
		return defaultProjectiles
	}
	return projectiles
}

func (p *Parser) parseUnits() ([]domain.Unit, error) {
	path := filepath.Join(p.GamePath, "export_descr_unit.txt")

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	unitTexts, err := p.parseUnitTexts()
	if err != nil {
		return nil, err
	}

	var units []domain.Unit
	var unit domain.Unit
	var isUnit bool

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		if i := strings.Index(line, ";"); i != -1 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		keyword := fields[0]

		switch keyword {
		case "type":
			unit = domain.Unit{}
			unit.Type = strings.Join(fields[1:], " ")
			isUnit = true
		case "ownership":
			for _, f := range fields[1:] {
				faction := trimComma(f)
				if faction != "" {
					unit.Ownership = append(unit.Ownership, faction)
				}
			}

			if text, ok := unitTexts[unit.Dictionary]; ok {
				unit.Name = text.Name
				unit.Descr = text.Descr
				unit.DescrShort = text.DescrShort
			}

			units = append(units, unit)
			isUnit = false
		default:
			if !isUnit {
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
				unit.VoiceType = fields[1]
			case "officer":
				unit.Officers = append(unit.Officers, strings.Join(fields[1:], " "))
			case "mount":
				unit.Mount = strings.Join(fields[1:], " ")
			case "mount_effect":
				unit.MountEffect = strings.Join(fields[1:], " ")
			case "soldier":
				vals := splitByComma(fields)
				unit.Soldier.Model = vals[0]
				unit.Soldier.Count = parseInt(vals[1])
				unit.Soldier.Extras = parseInt(vals[2])
				unit.Soldier.Mass = parseFloat(vals[3])
			case "attributes":
				for _, v := range splitByComma(fields) {
					unit.Attributes = append(unit.Attributes, v)
				}
			case "formation":
				unit.Formation = strings.Join(fields[1:], " ")
			case "stat_health":
				vals := splitByComma(fields)
				unit.StatHealth = [2]int{parseInt(vals[0]), parseInt(vals[1])}
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
				unit.StatPriArmour = domain.ArmourStats{
					Armour:   parseInt(fields[1]),
					DefSkill: parseInt(fields[2]),
					Shield:   parseInt(fields[3]),
					Sound:    trimComma(fields[4]),
				}
			case "stat_sec_armour":
				unit.StatSecArmour = domain.ArmourStats{
					Armour:   parseInt(fields[1]),
					DefSkill: parseInt(fields[2]),
					Sound:    trimComma(fields[3]),
				}
			case "stat_heat":
				unit.StatHeat = parseInt(fields[1])
			case "stat_ground":
				vals := splitByComma(fields)
				unit.StatGround = [4]int{
					parseInt(vals[0]), parseInt(vals[1]),
					parseInt(vals[2]), parseInt(vals[3]),
				}
			case "stat_mental":
				vals := splitByComma(fields)
				unit.StatMental = domain.MentalStats{
					Morale:     parseInt(vals[0]),
					Discipline: vals[1],
					Training:   vals[2],
				}
			case "stat_charge_dist":
				unit.StatChargeDist = parseInt(fields[1])
			case "stat_fire_delay":
				unit.StatFireDelay = parseInt(fields[1])
			case "stat_food":
				vals := splitByComma(fields)
				unit.StatFood = [2]int{parseInt(vals[0]), parseInt(vals[1])}
			case "stat_cost":
				vals := splitByComma(fields)
				unit.StatCost = domain.CostStats{
					Turns:         parseInt(vals[0]),
					Cost:          parseInt(vals[1]),
					Upkeep:        parseInt(vals[2]),
					WeaponUpgrade: parseInt(vals[3]),
					ArmourUpgrade: parseInt(vals[4]),
					Custom:        parseInt(vals[5]),
				}
			}
		}
	}

	return units, scanner.Err()
}

func parseWeaponStats(f []string) domain.WeaponStats {
	return domain.WeaponStats{
		Attack:      parseInt(f[0]),
		ChargeBonus: parseInt(f[1]),
		Missile:     f[2],
		Range:       parseInt(f[3]),
		Ammo:        parseInt(f[4]),
		WeaponType:  trimComma(f[5]),
		TechType:    trimComma(f[6]),
		DamageType:  trimComma(f[7]),
		SoundType:   trimComma(f[8]),
		MinDelay:    parseFloat(f[9]),
		Factor:      parseFloat(f[10]),
	}
}

// splitByComma объединяет всё после keyword и разбивает по запятой.
// Решает проблему когда значения написаны без пробелов: "0, 2,-6,-2"
func splitByComma(fields []string) []string {
	raw := strings.Join(fields[1:], " ")
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func trimComma(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), ",")
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(trimComma(s))
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(trimComma(s), 64)
	return v
}

func (p *Parser) parseBuildings() ([]domain.BuildingGroup, error) {
	path := filepath.Join(p.GamePath, "export_descr_buildings.txt")

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buildings []domain.BuildingGroup
	var currentBuilding domain.BuildingGroup
	var currentLevel domain.BuildingLevel

	depth := 0
	inCapability := false
	inUpgrades := false
	var pendingLevelName string
	var pendingLevelFactions []string
	var pendingLevelDependency *domain.BuildingDependency

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

		switch line {
		case "{":
			depth++
			if depth == 3 && pendingLevelName != "" {
				currentLevel = domain.BuildingLevel{
					Name:             pendingLevelName,
					RequiredCultures: pendingLevelFactions,
					Dependency:       pendingLevelDependency,
				}
				pendingLevelName = ""
				pendingLevelFactions = nil
				pendingLevelDependency = nil
			}
			continue
		case "}":
			if depth == 4 {
				inCapability = false
				inUpgrades = false
			} else if depth == 3 {
				currentBuilding.Levels = append(currentBuilding.Levels, currentLevel)
			} else if depth == 1 {
				buildings = append(buildings, currentBuilding)
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
			if fields[0] == "building" && len(fields) > 1 {
				currentBuilding = domain.BuildingGroup{Name: fields[1]}
			}
		case 2:
			if fields[0] != "levels" {
				pendingLevelName = fields[0]
				pendingLevelFactions = extractFactions(line)
				pendingLevelDependency = extractBuildingDependency(line)
			}
		case 3:
			switch fields[0] {
			case "capability":
				inCapability = true
			case "upgrades":
				inUpgrades = true
			case "construction":
				currentLevel.Construction = parseInt(fields[1])
			case "cost":
				currentLevel.Cost = parseInt(fields[1])
			case "settlement_min":
				currentLevel.SettlementMin = fields[1]
			case "building_present_min_level":
				if len(fields) >= 3 {
					currentLevel.Dependency = &domain.BuildingDependency{Group: fields[1], Level: fields[2]}
				}
			}
		case 4:
			if inCapability {
				if fields[0] == "recruit" {
					if slot, ok := parseRecruitLine(line); ok {
						currentLevel.RecruitSlots = append(currentLevel.RecruitSlots, slot)
					}
				} else {
					currentLevel.BonusLines = append(currentLevel.BonusLines, line)
				}
			} else if inUpgrades {
				currentLevel.Upgrades = append(currentLevel.Upgrades, trimComma(fields[0]))
			}
		}
	}

	return buildings, scanner.Err()
}

func extractBuildingDependency(line string) *domain.BuildingDependency {
	const marker = "building_present_min_level"
	idx := strings.Index(line, marker)
	if idx == -1 {
		return nil
	}
	parts := strings.Fields(line[idx:])
	if len(parts) < 3 {
		return nil
	}
	return &domain.BuildingDependency{Group: parts[1], Level: parts[2]}
}

func extractFactions(line string) []string {
	start := strings.Index(line, "{")
	end := strings.Index(line, "}")
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	var factions []string
	for _, f := range strings.Split(line[start+1:end], ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			factions = append(factions, f)
		}
	}
	return factions
}

func parseRecruitLine(line string) (domain.RecruitSlot, bool) {
	start := strings.Index(line, "\"")
	end := strings.LastIndex(line, "\"")
	if start == -1 || start == end {
		return domain.RecruitSlot{}, false
	}

	unitType := line[start+1 : end]
	rest := strings.TrimSpace(line[end+1:])

	level := 0
	if restFields := strings.Fields(rest); len(restFields) > 0 {
		level, _ = strconv.Atoi(restFields[0])
	}

	factStart := strings.Index(rest, "{")
	factEnd := strings.Index(rest, "}")

	var factions []string
	var conditions string
	if factStart != -1 && factEnd != -1 && factEnd > factStart {
		for _, f := range strings.Split(rest[factStart+1:factEnd], ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				factions = append(factions, f)
			}
		}
		conditions = strings.TrimSpace(rest[factEnd+1:])
	}

	return domain.RecruitSlot{
		UnitType:     unitType,
		Level:        level,
		Requirements: factions,
		Conditions:   conditions,
	}, true
}

func (p *Parser) parseFactions() ([]domain.Faction, error) {
	path := filepath.Join(p.GamePath, "descr_sm_factions.txt")

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var factions []domain.Faction
	var current domain.Faction
	var inBlock bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			if inBlock && current.Name != "" {
				factions = append(factions, current)
				current = domain.Faction{}
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
			if inBlock && current.Name != "" {
				factions = append(factions, current)
				current = domain.Faction{}
			}
			current.Name = fields[1]
			inBlock = true
		case "culture":
			current.Culture = fields[1]
		case "primary_colour":
			current.PrimaryColour = parseColour(fields[1:])
		case "secondary_colour":
			current.SecondaryColour = parseColour(fields[1:])
		}
	}

	if inBlock && current.Name != "" {
		factions = append(factions, current)
	}

	return factions, scanner.Err()
}

func parseColour(fields []string) domain.Colour {
	var c domain.Colour
	for i := 0; i+1 < len(fields); i += 2 {
		val, _ := strconv.Atoi(fields[i+1])
		switch fields[i] {
		case "red":
			c.R = uint8(val)
		case "green":
			c.G = uint8(val)
		case "blue":
			c.B = uint8(val)
		}
	}
	return c
}

func (p *Parser) parseUnitTexts() (map[string]unitText, error) {
	path := filepath.Join(p.GamePath, "text", "export_units.txt")

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
	scanner := bufio.NewScanner(transform.NewReader(f, decoder))

	result := make(map[string]unitText)

	var currentKey string
	var currentField string
	var expectValue bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "¬") {
			continue
		}

		if strings.HasPrefix(line, "{") {
			end := strings.Index(line, "}")
			if end == -1 {
				continue
			}
			tag := line[1:end]
			rest := strings.TrimSpace(line[end+1:])

			switch {
			case strings.HasSuffix(tag, "_descr_short"):
				currentKey = strings.TrimSuffix(tag, "_descr_short")
				currentField = "descr_short"
			case strings.HasSuffix(tag, "_descr"):
				currentKey = strings.TrimSuffix(tag, "_descr")
				currentField = "descr"
			default:
				currentKey = tag
				currentField = "name"
			}

			if rest != "" {
				setUnitTextValue(result, currentKey, currentField, rest)
				expectValue = false
			} else {
				expectValue = true
			}
		} else if expectValue {
			setUnitTextValue(result, currentKey, currentField, line)
			expectValue = false
		}
	}

	return result, scanner.Err()
}

func setUnitTextValue(m map[string]unitText, key, field, value string) {
	entry := m[key]
	switch field {
	case "name":
		entry.Name = value
	case "descr":
		entry.Descr = value
	case "descr_short":
		entry.DescrShort = value
	}
	m[key] = entry
}

// parseFactionDisplayNames reads all UTF-16 text files in data/text/ and collects
// {KEY}\tValue entries, returning a map of uppercase key → display name.
// This covers faction names stored in files like expanded_bi.txt.
func (p *Parser) parseFactionDisplayNames() map[string]string {
	result := make(map[string]string)
	textDir := filepath.Join(p.GamePath, "text")
	entries, err := os.ReadDir(textDir)
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".txt") {
			continue
		}
		p.parseTextFileIntoMap(filepath.Join(textDir, entry.Name()), result)
	}
	return result
}

func (p *Parser) parseTextFileIntoMap(path string, out map[string]string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	decoder := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
	scanner := bufio.NewScanner(transform.NewReader(f, decoder))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "{") {
			continue
		}
		end := strings.Index(line, "}")
		if end == -1 {
			continue
		}
		key := line[1:end]
		value := strings.TrimSpace(line[end+1:])
		if value != "" && !strings.HasPrefix(value, "¬") {
			out[key] = value
		}
	}
}

func (p *Parser) findFile(pattern string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(p.GamePath, pattern))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("file not found: %s", pattern)
	}
	return matches[0], nil
}
