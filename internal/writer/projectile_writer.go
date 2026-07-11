package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"tw-forge/internal/domain"
	"tw-forge/internal/logger"
	"tw-forge/internal/repository"
)

const projectileFileName = "descr_projectile.txt"
const projectileNewFileName = "descr_projectile_new.txt"

// projectileSourceFileName returns which of the two filenames the backup holds.
// RTW's Battle for Rome/Alexander expansions use descr_projectile_new.txt;
// M2TW and base RTW use descr_projectile.txt. Same lookup rule as the parser.
func (w *GameWriter) projectileSourceFileName() string {
	if _, err := os.Stat(filepath.Join(w.backupPath, projectileNewFileName)); err == nil {
		return projectileNewFileName
	}
	if _, err := os.Stat(filepath.Join(w.gamePath, projectileNewFileName)); err == nil {
		return projectileNewFileName
	}
	return projectileFileName
}

// SaveProjectilesDraft rewrites descr_projectile.txt into the draft folder,
// patching only the blocks named in changes. Untouched entries are copied
// byte-for-byte from the backed-up original.
func (w *GameWriter) SaveProjectilesDraft(file domain.ProjectileFile, changes map[string]repository.ChangeType) error {
	if err := w.ensureInit(); err != nil {
		return err
	}

	filename := w.projectileSourceFileName()
	src := filepath.Join(w.backupPath, filename)
	raw, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // file doesn't exist in this game version, nothing to patch
		}
		return fmt.Errorf("open source %s: %w", filename, err)
	}

	byName := make(map[string]domain.Projectile, len(file.Projectiles))
	for _, p := range file.Projectiles {
		byName[p.Name] = p
	}

	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(content, "\n")

	var out []string
	headerIdx := -1
	flushBlock := func(end int) {
		if headerIdx == -1 {
			out = append(out, lines[:end]...)
			return
		}
		blockName := headerName(lines[headerIdx])
		changeType, changed := changes[blockName]
		if !changed {
			out = append(out, lines[headerIdx:end]...)
			return
		}
		if changeType == repository.ChangeDeleted {
			return // drop the block entirely
		}
		p, ok := byName[blockName]
		if !ok {
			out = append(out, lines[headerIdx:end]...)
			return
		}
		out = append(out, strings.Split(serializeProjectileBlock(p), "\n")...)
	}

	for i, line := range lines {
		if headerName(line) != "" {
			flushBlock(i)
			headerIdx = i
		}
	}
	flushBlock(len(lines))

	// Appended (newly created) projectiles.
	for name, ct := range changes {
		if ct != repository.ChangeAdded {
			continue
		}
		p, ok := byName[name]
		if !ok {
			continue
		}
		out = append(out, "", "")
		out = append(out, strings.Split(serializeProjectileBlock(p), "\n")...)
	}

	dst := filepath.Join(w.draftPath, filename)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create draft dir: %w", err)
	}
	if err := os.WriteFile(dst, []byte(strings.Join(out, "\r\n")), 0644); err != nil {
		return err
	}
	logger.FileWrite(dst, "write", filename, 0)
	return nil
}

// headerName returns the projectile name if line is a "projectile <name>" header, else "".
func headerName(line string) string {
	key, fields := headerKeyAndFields(line)
	if key != "projectile" || len(fields) < 2 {
		return ""
	}
	return fields[1]
}

func headerKeyAndFields(line string) (string, []string) {
	line = strings.TrimSpace(line)
	if i := strings.Index(line, ";"); i != -1 {
		line = strings.TrimSpace(line[:i])
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields
}

// serializeProjectileBlock patches p.RawBlock in place: known ballistics fields
// are replaced or inserted, everything else (effects, display, models, comments,
// rocket/trail params) is left untouched.
func serializeProjectileBlock(p domain.Projectile) string {
	lines := strings.Split(p.RawBlock, "\n")

	lines = setLine(lines, "projectile", "projectile "+p.Name)

	headerAnchor := 0
	if p.FlamingOf != "" {
		lines, headerAnchor = upsertLine(lines, "flaming", formatLine("flaming", p.FlamingOf), headerAnchor)
	} else {
		lines = removeLine(lines, "flaming")
	}
	if p.ExplodingOf != "" {
		lines, headerAnchor = upsertLine(lines, "exploding", formatLine("exploding", p.ExplodingOf), headerAnchor)
	} else {
		lines = removeLine(lines, "exploding")
	}

	effectFields := []struct {
		key   string
		value string
	}{
		{"effect", p.Effect},
		{"end_effect", p.EndEffect},
		{"end_man_effect", p.EndManEffect},
		{"end_package_effect", p.EndPackageEffect},
		{"end_shatter_effect", p.EndShatterEffect},
		{"end_shatter_man_effect", p.EndShatterManEffect},
		{"end_shatter_package_effect", p.EndShatterPackageEffect},
	}
	for _, ef := range effectFields {
		if ef.value != "" {
			lines, headerAnchor = upsertLine(lines, ef.key, formatLine(ef.key, ef.value), headerAnchor)
		} else {
			lines = removeLine(lines, ef.key)
		}
	}

	lines = setLine(lines, "damage", formatLine("damage", strconv.Itoa(p.Damage)))
	damageAnchor := indexOfKey(lines, "damage")
	if damageAnchor == -1 {
		damageAnchor = len(lines) - 1
	}

	if p.DamageToTroops != nil {
		lines, damageAnchor = upsertLine(lines, "damage_to_troops", formatLine("damage_to_troops", strconv.Itoa(*p.DamageToTroops)), damageAnchor)
	} else {
		lines = removeLine(lines, "damage_to_troops")
	}

	lines = setLine(lines, "radius", formatLine("radius", rtwFloat(p.Radius)))
	lines = setLine(lines, "mass", formatLine("mass", rtwFloat(p.Mass)))

	if p.Area != nil {
		lines, damageAnchor = upsertLine(lines, "area", formatLine("area", rtwFloat(*p.Area)), damageAnchor)
	} else {
		lines = removeLine(lines, "area")
	}

	if p.AccuracyVsUnits != nil {
		lines, damageAnchor = upsertLine(lines, "accuracy_vs_units", formatLine("accuracy_vs_units", rtwFloat(*p.AccuracyVsUnits)), damageAnchor)
	} else {
		lines = removeLine(lines, "accuracy_vs_units")
	}
	if p.AccuracyVsBuildings != nil {
		lines, damageAnchor = upsertLine(lines, "accuracy_vs_buildings", formatLine("accuracy_vs_buildings", rtwFloat(*p.AccuracyVsBuildings)), damageAnchor)
	} else {
		lines = removeLine(lines, "accuracy_vs_buildings")
	}
	if p.AccuracyVsTowers != nil {
		lines, _ = upsertLine(lines, "accuracy_vs_towers", formatLine("accuracy_vs_towers", rtwFloat(*p.AccuracyVsTowers)), damageAnchor)
	} else {
		lines = removeLine(lines, "accuracy_vs_towers")
	}

	lines = setLine(lines, "min_angle", formatLine("min_angle", strconv.Itoa(p.MinAngle)))
	lines = setLine(lines, "max_angle", formatLine("max_angle", strconv.Itoa(p.MaxAngle)))
	lines = setLine(lines, "velocity", formatLine("velocity", joinFloats(p.Velocity)))

	velocityAnchor := indexOfKey(lines, "velocity")
	if velocityAnchor == -1 {
		velocityAnchor = len(lines) - 1
	}
	present := make(map[string]bool, len(p.Flags))
	for _, f := range p.Flags {
		present[f] = true
	}
	for _, flag := range domain.ManagedProjectileFlags {
		if present[flag] {
			lines, velocityAnchor = upsertLine(lines, flag, flag, velocityAnchor)
		} else {
			lines = removeLine(lines, flag)
		}
	}

	return strings.Join(lines, "\n")
}

// formatLine mirrors the alignment convention used for EDU: pad short keys to a
// fixed column, otherwise fall back to a single separating space (see the
// recruit_priority_offset bug fixed in serializer_m2tw.go).
func formatLine(key, value string) string {
	if len(key) < 22 {
		return fmt.Sprintf("%-22s%s", key, value)
	}
	return key + " " + value
}

func joinFloats(vals []float64) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = rtwFloat(v)
	}
	return strings.Join(parts, " ")
}

func firstToken(line string) string {
	key, _ := headerKeyAndFields(line)
	return key
}

func indexOfKey(lines []string, key string) int {
	for i, l := range lines {
		if firstToken(l) == key {
			return i
		}
	}
	return -1
}

// setLine replaces the line whose first token is key, or appends newLine if not found.
func setLine(lines []string, key, newLine string) []string {
	for i, l := range lines {
		if firstToken(l) == key {
			lines[i] = newLine
			return lines
		}
	}
	return append(lines, newLine)
}

// upsertLine replaces the line whose first token is key, or inserts newLine right
// after anchorIdx when the key is absent. Returns the updated slice and the index
// of the (possibly newly inserted) line, so callers can chain further inserts
// after the same anchor point.
func upsertLine(lines []string, key, newLine string, anchorIdx int) ([]string, int) {
	for i, l := range lines {
		if firstToken(l) == key {
			lines[i] = newLine
			return lines, i
		}
	}
	if anchorIdx < 0 || anchorIdx >= len(lines) {
		anchorIdx = len(lines) - 1
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:anchorIdx+1]...)
	out = append(out, newLine)
	out = append(out, lines[anchorIdx+1:]...)
	return out, anchorIdx + 1
}

func removeLine(lines []string, key string) []string {
	out := lines[:0]
	for _, l := range lines {
		if firstToken(l) == key {
			continue
		}
		out = append(out, l)
	}
	return out
}
