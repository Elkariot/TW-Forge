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

// mercenariesRelPath returns campaign's descr_mercenaries.txt path relative to gamePath.
func mercenariesRelPath(campaign string) string {
	return filepath.Join("world", "maps", "campaign", campaign, "descr_mercenaries.txt")
}

// SaveMercenariesDraft rewrites campaign's descr_mercenaries.txt into the draft folder,
// patching only the pools named in changes. Untouched pools are copied byte-for-byte
// from the backed-up original, same pattern as SaveProjectilesDraft.
func (w *GameWriter) SaveMercenariesDraft(campaign string, file domain.MercenaryFile, changes map[string]repository.ChangeType) error {
	if err := w.ensureInit(); err != nil {
		return err
	}

	relPath := mercenariesRelPath(campaign)
	backupFile := filepath.Join(w.backupPath, relPath)
	if _, err := os.Stat(backupFile); err != nil {
		src := filepath.Join(w.gamePath, relPath)
		if _, err := os.Stat(src); err != nil {
			// Most mods don't ship their own campaign map — fall back to the base
			// game's copy so saving still works; Apply() then writes the result into
			// the mod's own tree, creating a proper mod-side override.
			if w.baseGameDataPath == "" {
				return nil
			}
			src = filepath.Join(w.baseGameDataPath, relPath)
			if _, err := os.Stat(src); err != nil {
				return nil // not present in the mod or the base game
			}
		}
		if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
			return fmt.Errorf("create backup dir: %w", err)
		}
		if err := copyFile(src, backupFile); err != nil {
			return fmt.Errorf("backup %s: %w", relPath, err)
		}
	}

	raw, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("open source %s: %w", relPath, err)
	}

	byName := make(map[string]domain.MercenaryPool, len(file.Pools))
	for _, p := range file.Pools {
		byName[p.Name] = p
	}

	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(content, "\n")

	var out []string
	poolStart := -1
	flushBlock := func(end int) {
		if poolStart == -1 {
			out = append(out, lines[:end]...)
			return
		}
		name := mercPoolHeaderName(lines[poolStart])
		changeType, changed := changes[name]
		if !changed {
			out = append(out, lines[poolStart:end]...)
			return
		}
		if changeType == repository.ChangeDeleted {
			return // drop the block entirely
		}
		p, ok := byName[name]
		if !ok {
			out = append(out, lines[poolStart:end]...)
			return
		}
		out = append(out, strings.Split(formatMercenaryPool(p), "\n")...)
	}

	for i, line := range lines {
		if mercPoolHeaderName(line) != "" {
			flushBlock(i)
			poolStart = i
		}
	}
	flushBlock(len(lines))

	// Appended (newly created) pools.
	for name, ct := range changes {
		if ct != repository.ChangeAdded {
			continue
		}
		p, ok := byName[name]
		if !ok {
			continue
		}
		out = append(out, "", "")
		out = append(out, strings.Split(formatMercenaryPool(p), "\n")...)
	}

	dst := filepath.Join(w.draftPath, relPath)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create draft dir: %w", err)
	}
	if err := os.WriteFile(dst, []byte(strings.Join(out, "\r\n")), 0644); err != nil {
		return err
	}
	logger.FileWrite(dst, "write", relPath, 0)
	return nil
}

// mercPoolHeaderName returns the pool name if line is a "pool <Name>" header, else "".
func mercPoolHeaderName(line string) string {
	t := strings.TrimSpace(line)
	if i := strings.Index(t, ";"); i != -1 {
		t = strings.TrimSpace(t[:i])
	}
	fields := strings.Fields(t)
	if len(fields) < 2 || fields[0] != "pool" {
		return ""
	}
	return fields[1]
}

// formatMercenaryPool serializes a pool's regions/unit lines fresh from typed fields —
// used only for pools created or edited via the UI; untouched pools keep their
// original RawBlock text (see SaveMercenariesDraft).
func formatMercenaryPool(p domain.MercenaryPool) string {
	var sb strings.Builder
	sb.WriteString("pool ")
	sb.WriteString(p.Name)
	sb.WriteString("\n\tregions ")
	sb.WriteString(strings.Join(p.Regions, " "))
	for _, u := range p.Units {
		sb.WriteString("\n")
		sb.WriteString(formatMercenaryUnit(u))
	}
	return sb.String()
}

func formatMercenaryUnit(u domain.MercenaryUnit) string {
	var sb strings.Builder
	sb.WriteString("\tunit ")
	sb.WriteString(u.Name)
	sb.WriteString("\t\t\t\texp ")
	sb.WriteString(strconv.Itoa(u.Exp))
	if u.Armour != nil {
		fmt.Fprintf(&sb, " armour %d", *u.Armour)
	}
	if u.WeaponLvl != nil {
		fmt.Fprintf(&sb, " weapon_lvl %d", *u.WeaponLvl)
	}
	fmt.Fprintf(&sb, " cost %d replenish %s - %s max %d initial %d",
		u.Cost, rtwFloat(u.ReplenishLow), rtwFloat(u.ReplenishHigh), u.Max, u.Initial)
	if u.EndYear != nil {
		fmt.Fprintf(&sb, " end_year %d", *u.EndYear)
	}
	if u.StartYear != nil {
		fmt.Fprintf(&sb, " start_year %d", *u.StartYear)
	}
	if len(u.Religions) > 0 {
		fmt.Fprintf(&sb, " religions { %s }", strings.Join(u.Religions, " "))
	}
	if u.Crusading {
		sb.WriteString(" crusading")
	}
	if len(u.Events) > 0 {
		fmt.Fprintf(&sb, " events { %s }", strings.Join(u.Events, " "))
	}
	return sb.String()
}

// applyMercenariesCampaigns copies every drafted campaign's descr_mercenaries.txt back
// over the game files. Draft/backup layout mirrors the real world/maps/campaign/ tree,
// so any number of campaigns touched in a session are picked up generically.
func (w *GameWriter) applyMercenariesCampaigns() error {
	return w.applyDirRecursive(filepath.Join("world", "maps", "campaign"))
}

// restoreMercenariesCampaigns copies every backed-up campaign's descr_mercenaries.txt
// back over the game files, undoing Applied changes (see RestoreFromBackup).
func (w *GameWriter) restoreMercenariesCampaigns() error {
	root := filepath.Join(w.backupPath, "world", "maps", "campaign")
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(w.backupPath, path)
		if relErr != nil {
			return relErr
		}
		return copyFile(path, filepath.Join(w.gamePath, rel))
	})
}
