package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"tw-forge/internal/domain"
)

var boolFlagKeywords = func() map[string]bool {
	m := make(map[string]bool, len(domain.ManagedProjectileFlags))
	for _, f := range domain.ManagedProjectileFlags {
		m[f] = true
	}
	return m
}()

// ParseProjectileFile parses descr_projectile.txt (RTW uses descr_projectile_new.txt
// when present, same format as M2TW's descr_projectile.txt).
func ParseProjectileFile(gamePath string) (*domain.ProjectileFile, error) {
	path := filepath.Join(gamePath, "descr_projectile_new.txt")
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join(gamePath, "descr_projectile.txt")
	}
	return parseProjectileFile(path)
}

func parseProjectileFile(path string) (*domain.ProjectileFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(content, "\n")

	result := &domain.ProjectileFile{}

	headerIdx := -1
	for i, line := range lines {
		key, fields := lineKeyAndFields(line)
		switch key {
		case "delay":
			if len(fields) >= 3 {
				result.Delays = append(result.Delays, domain.ProjectileDelay{
					Type:    fields[1],
					Seconds: parseFloat(fields[2]),
				})
			}
		case "projectile":
			if headerIdx != -1 {
				result.Projectiles = append(result.Projectiles, buildProjectile(lines[headerIdx:i]))
			}
			headerIdx = i
		}
	}
	if headerIdx != -1 {
		result.Projectiles = append(result.Projectiles, buildProjectile(lines[headerIdx:]))
	}

	return result, nil
}

// buildProjectile trims trailing blank lines from block (kept up to the next
// "projectile" header by the caller) and extracts typed fields from it.
func buildProjectile(block []string) domain.Projectile {
	for len(block) > 0 && strings.TrimSpace(block[len(block)-1]) == "" {
		block = block[:len(block)-1]
	}

	p := domain.Projectile{RawBlock: strings.Join(block, "\n")}

	scanner := bufio.NewScanner(strings.NewReader(p.RawBlock))
	for scanner.Scan() {
		key, fields := lineKeyAndFields(scanner.Text())
		if key == "" {
			continue
		}

		switch key {
		case "projectile":
			if len(fields) >= 2 {
				p.Name = fields[1]
			}
		case "flaming":
			if len(fields) >= 2 {
				p.FlamingOf = fields[1]
			}
		case "exploding":
			if len(fields) >= 2 {
				p.ExplodingOf = fields[1]
			}
		case "effect":
			if len(fields) >= 2 {
				p.Effect = fields[1]
			}
		case "end_effect":
			if len(fields) >= 2 {
				p.EndEffect = fields[1]
			}
		case "end_man_effect":
			if len(fields) >= 2 {
				p.EndManEffect = fields[1]
			}
		case "end_package_effect":
			if len(fields) >= 2 {
				p.EndPackageEffect = fields[1]
			}
		case "end_shatter_effect":
			if len(fields) >= 2 {
				p.EndShatterEffect = fields[1]
			}
		case "end_shatter_man_effect":
			if len(fields) >= 2 {
				p.EndShatterManEffect = fields[1]
			}
		case "end_shatter_package_effect":
			if len(fields) >= 2 {
				p.EndShatterPackageEffect = fields[1]
			}
		case "damage":
			if len(fields) >= 2 {
				p.Damage = parseInt(fields[1])
			}
		case "damage_to_troops":
			if len(fields) >= 2 {
				v := parseInt(fields[1])
				p.DamageToTroops = &v
			}
		case "radius":
			if len(fields) >= 2 {
				p.Radius = parseFloat(fields[1])
			}
		case "mass":
			if len(fields) >= 2 {
				p.Mass = parseFloat(fields[1])
			}
		case "area":
			if len(fields) >= 2 {
				v := parseFloat(fields[1])
				p.Area = &v
			}
		case "accuracy_vs_units":
			if len(fields) >= 2 {
				v := parseFloat(fields[1])
				p.AccuracyVsUnits = &v
			}
		case "accuracy_vs_buildings":
			if len(fields) >= 2 {
				v := parseFloat(fields[1])
				p.AccuracyVsBuildings = &v
			}
		case "accuracy_vs_towers":
			if len(fields) >= 2 {
				v := parseFloat(fields[1])
				p.AccuracyVsTowers = &v
			}
		case "min_angle":
			if len(fields) >= 2 {
				p.MinAngle = parseInt(fields[1])
			}
		case "max_angle":
			if len(fields) >= 2 {
				p.MaxAngle = parseInt(fields[1])
			}
		case "velocity":
			p.Velocity = parseNumberList(fields[1:])
		default:
			if boolFlagKeywords[key] && len(fields) == 1 {
				p.Flags = append(p.Flags, key)
			}
		}
	}

	return p
}

// lineKeyAndFields strips an inline ";" comment and returns the first whitespace
// token (lowercase keyword) plus the full field list of the remaining line.
// Blank and comment-only lines return an empty key.
func lineKeyAndFields(line string) (string, []string) {
	line = strings.TrimSpace(line)
	if i := strings.Index(line, ";"); i != -1 {
		line = strings.TrimSpace(line[:i])
	}
	if line == "" {
		return "", nil
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields
}

// parseNumberList joins the remaining fields and splits on both comma and
// whitespace, so "55, 30" and "20 42" parse the same way.
func parseNumberList(fields []string) []float64 {
	joined := strings.ReplaceAll(strings.Join(fields, " "), ",", " ")
	var result []float64
	for f := range strings.FieldsSeq(joined) {
		result = append(result, parseFloat(f))
	}
	return result
}
