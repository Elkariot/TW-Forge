package parser

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"tw-forge/internal/domain"
)

// mercUnitKeywords are the recognized field keywords on a "unit ..." line. Everything
// between "unit" and the first of these tokens is the unit name (which may itself
// contain spaces or a trailing comma, e.g. "Native Mercenaries", "merc cog,").
var mercUnitKeywords = map[string]bool{
	"exp": true, "armour": true, "weapon_lvl": true, "cost": true,
	"replenish": true, "max": true, "initial": true,
	"end_year": true, "start_year": true, "religions": true,
	"crusading": true, "events": true,
}

// ListMercenaryCampaigns returns campaign folder names (relative to
// world/maps/campaign/, e.g. "imperial_campaign", "custom/Fourth_Era") that have a
// descr_mercenaries.txt file. gamePath is the game's data folder.
func ListMercenaryCampaigns(gamePath string) []string {
	root := filepath.Join(gamePath, "world", "maps", "campaign")
	var campaigns []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.EqualFold(d.Name(), "descr_mercenaries.txt") {
			return nil
		}
		rel, relErr := filepath.Rel(root, filepath.Dir(path))
		if relErr == nil {
			campaigns = append(campaigns, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(campaigns)
	return campaigns
}

// MercenariesFilePath returns the path to a campaign's descr_mercenaries.txt.
func MercenariesFilePath(gamePath, campaign string) string {
	return filepath.Join(gamePath, "world", "maps", "campaign", campaign, "descr_mercenaries.txt")
}

// RegionsFilePath returns the descr_regions.txt that applies to campaign: a custom
// campaign map (e.g. TES's Fourth_Era) ships its own copy next to descr_mercenaries.txt;
// otherwise regions are defined once for every campaign in world/maps/base/.
func RegionsFilePath(gamePath, campaign string) string {
	perCampaign := filepath.Join(gamePath, "world", "maps", "campaign", campaign, "descr_regions.txt")
	if _, err := os.Stat(perCampaign); err == nil {
		return perCampaign
	}
	return filepath.Join(gamePath, "world", "maps", "base", "descr_regions.txt")
}

// ParseRegionNames extracts region names from a descr_regions.txt: every non-blank,
// non-comment, non-indented line starts a new region and its first token is the name.
func ParseRegionNames(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")

	var regions []string
	for _, line := range strings.Split(content, "\n") {
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, ";") {
			continue
		}
		if fields := strings.Fields(t); len(fields) > 0 {
			regions = append(regions, fields[0])
		}
	}
	return regions, nil
}

// ParseMercenaryFile parses a campaign's descr_mercenaries.txt.
func ParseMercenaryFile(path string) (*domain.MercenaryFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(content, "\n")

	file := &domain.MercenaryFile{}

	headerEnd := 0
	for headerEnd < len(lines) {
		t := strings.TrimSpace(lines[headerEnd])
		if t != "" && !strings.HasPrefix(t, ";") {
			break
		}
		headerEnd++
	}
	file.Header = strings.Join(lines[:headerEnd], "\n")

	poolStart := -1
	flush := func(end int) {
		if poolStart == -1 {
			return
		}
		file.Pools = append(file.Pools, buildMercenaryPool(lines[poolStart:end]))
	}
	for i := headerEnd; i < len(lines); i++ {
		key, _ := mercLineKeyAndFields(lines[i])
		if key == "pool" {
			flush(i)
			poolStart = i
		}
	}
	flush(len(lines))

	return file, nil
}

func buildMercenaryPool(block []string) domain.MercenaryPool {
	for len(block) > 0 && strings.TrimSpace(block[len(block)-1]) == "" {
		block = block[:len(block)-1]
	}

	p := domain.MercenaryPool{RawBlock: strings.Join(block, "\n")}
	for _, line := range block {
		key, fields := mercLineKeyAndFields(line)
		switch key {
		case "pool":
			if len(fields) >= 2 {
				p.Name = fields[1]
			}
		case "regions":
			p.Regions = append(p.Regions, fields[1:]...)
		case "unit":
			p.Units = append(p.Units, parseMercUnitLine(fields))
		}
	}
	return p
}

func parseMercUnitLine(fields []string) domain.MercenaryUnit {
	i := 1 // fields[0] == "unit"
	var nameParts []string
	for i < len(fields) && !mercUnitKeywords[fields[i]] {
		nameParts = append(nameParts, fields[i])
		i++
	}
	u := domain.MercenaryUnit{Name: strings.Join(nameParts, " ")}

	for i < len(fields) {
		switch fields[i] {
		case "exp":
			if i+1 < len(fields) {
				u.Exp = mercAtoi(fields[i+1])
			}
			i += 2
		case "armour":
			if i+1 < len(fields) {
				v := mercAtoi(fields[i+1])
				u.Armour = &v
			}
			i += 2
		case "weapon_lvl":
			if i+1 < len(fields) {
				v := mercAtoi(fields[i+1])
				u.WeaponLvl = &v
			}
			i += 2
		case "cost":
			if i+1 < len(fields) {
				u.Cost = mercAtoi(fields[i+1])
			}
			i += 2
		case "replenish":
			// "replenish <low> - <high>"
			if i+3 < len(fields) {
				u.ReplenishLow = mercAtof(fields[i+1])
				u.ReplenishHigh = mercAtof(fields[i+3])
			}
			i += 4
		case "max":
			if i+1 < len(fields) {
				u.Max = mercAtoi(fields[i+1])
			}
			i += 2
		case "initial":
			if i+1 < len(fields) {
				u.Initial = mercAtoi(fields[i+1])
			}
			i += 2
		case "end_year":
			if i+1 < len(fields) {
				v := mercAtoi(fields[i+1])
				u.EndYear = &v
			}
			i += 2
		case "start_year":
			if i+1 < len(fields) {
				v := mercAtoi(fields[i+1])
				u.StartYear = &v
			}
			i += 2
		case "crusading":
			u.Crusading = true
			i++
		case "religions":
			i += 2 // "religions" "{"
			for i < len(fields) && fields[i] != "}" {
				u.Religions = append(u.Religions, fields[i])
				i++
			}
			i++ // "}"
		case "events":
			i += 2 // "events" "{"
			for i < len(fields) && fields[i] != "}" {
				u.Events = append(u.Events, fields[i])
				i++
			}
			i++ // "}"
		default:
			i++
		}
	}
	return u
}

// mercLineKeyAndFields strips an inline ";" comment and returns the first whitespace
// token (lowercase keyword) plus the full field list of the remaining line.
func mercLineKeyAndFields(line string) (string, []string) {
	t := strings.TrimSpace(line)
	if i := strings.Index(t, ";"); i != -1 {
		t = strings.TrimSpace(t[:i])
	}
	if t == "" {
		return "", nil
	}
	fields := strings.Fields(t)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields
}

func mercAtoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func mercAtof(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
