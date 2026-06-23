package writer

import (
	"fmt"
	"strconv"
	"strings"
	"tw-forge/internal/domain"
	"tw-forge/internal/parser"
	"os"
	"path/filepath"
)

// CopyBattleModelDraft copies a model entry from srcName to dstName,
// writing the result into the draft directory.
// Reads from the existing draft if present, otherwise from the game directory.
// Best-effort: if srcName is not found in the modeldb, returns an error.
func (w *GameWriter) CopyBattleModelDraft(srcName, dstName string) error {
	if err := w.ensureInit(); err != nil {
		return err
	}

	relPath := w.modelDBRelPath()
	gameModelDB := filepath.Join(w.gamePath, relPath)
	draftModelDB := filepath.Join(w.draftPath, relPath)

	if _, err := os.Stat(gameModelDB); err != nil {
		return fmt.Errorf("battle_models.modeldb not found in game directory")
	}

	backupModelDB := filepath.Join(w.backupPath, relPath)
	if _, err := os.Stat(backupModelDB); err != nil {
		_ = os.MkdirAll(filepath.Dir(backupModelDB), 0755)
		if err := copyFile(gameModelDB, backupModelDB); err != nil {
			return fmt.Errorf("backup battle_models.modeldb: %w", err)
		}
	}

	loadPath := gameModelDB
	if _, err := os.Stat(draftModelDB); err == nil {
		loadPath = draftModelDB
	}

	db, err := parser.ParseBattleModels(loadPath)
	if err != nil {
		return fmt.Errorf("parse battle_models.modeldb: %w", err)
	}

	newDB, err := CopyModelEntry(db, srcName, dstName)
	if err != nil {
		return err
	}

	return WriteBattleModels(newDB, draftModelDB)
}

// PatchSoldierFactionDraft ensures the soldier model entry in battle_models.modeldb
// has texture entries for dstFaction. Copies them from srcFaction (or the first
// available faction if srcFaction is absent). No-op when factions are equal or
// when the entry doesn't exist.
func (w *GameWriter) PatchSoldierFactionDraft(soldierModel, srcFaction, dstFaction string) error {
	if soldierModel == "" || srcFaction == "" || dstFaction == "" || srcFaction == dstFaction {
		return nil
	}
	if err := w.ensureInit(); err != nil {
		return err
	}

	relPath := w.modelDBRelPath()
	gameModelDB := filepath.Join(w.gamePath, relPath)
	draftModelDB := filepath.Join(w.draftPath, relPath)

	if _, err := os.Stat(gameModelDB); err != nil {
		return nil
	}

	backupModelDB := filepath.Join(w.backupPath, relPath)
	if _, err := os.Stat(backupModelDB); err != nil {
		_ = os.MkdirAll(filepath.Dir(backupModelDB), 0755)
		_ = copyFile(gameModelDB, backupModelDB)
	}

	loadPath := gameModelDB
	if _, err := os.Stat(draftModelDB); err == nil {
		loadPath = draftModelDB
	}

	db, err := parser.ParseBattleModels(loadPath)
	if err != nil {
		return err
	}

	idx, ok := db.Index[soldierModel]
	if !ok {
		return nil
	}

	newBlock, changed := addFactionToRawBlock(db.Models[idx].RawBlock, srcFaction, dstFaction)
	if !changed {
		return nil
	}

	db.Models[idx].RawBlock = newBlock
	if err := os.MkdirAll(filepath.Dir(draftModelDB), 0755); err != nil {
		return err
	}
	return WriteBattleModels(db, draftModelDB)
}

// addFactionToRawBlock scans each faction-set in the raw model block and, for
// every set that contains srcFaction but not dstFaction, appends a dstFaction
// entry (same texture paths) and increments the set's count.
func addFactionToRawBlock(block, srcFaction, dstFaction string) (string, bool) {
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	changed := false

	isFac := func(s string) (string, bool) {
		t := strings.TrimSpace(s)
		p := strings.Fields(t)
		if len(p) != 2 {
			return "", false
		}
		n, e := strconv.Atoi(p[0])
		if e != nil || n <= 0 || n > 25 || len(p[1]) != n {
			return "", false
		}
		if strings.ContainsAny(p[1], "./0123456789") {
			return "", false
		}
		return p[1], true
	}

	isSingleInt := func(s string) (int, bool) {
		t := strings.TrimSpace(s)
		if strings.ContainsAny(t, " \t") {
			return 0, false
		}
		n, e := strconv.Atoi(t)
		return n, e == nil && n >= 0
	}

	type entry struct {
		name  string
		lines []string
	}

	out := make([]string, 0, len(lines)+20)
	i := 0

	for i < len(lines) {
		count, isCount := isSingleInt(lines[i])
		if !isCount || count <= 0 || count > 20 {
			out = append(out, lines[i])
			i++
			continue
		}

		// First non-empty lookahead must be a faction name
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j >= len(lines) {
			out = append(out, lines[i])
			i++
			continue
		}
		if _, ok := isFac(lines[j]); !ok {
			out = append(out, lines[i])
			i++
			continue
		}

		// Parse all faction entries in this set
		var entries []entry
		j = i + 1
		for j < len(lines) {
			fname, ok := isFac(lines[j])
			if !ok {
				break
			}
			e := entry{name: fname, lines: []string{lines[j]}}
			j++
			for j < len(lines) {
				if _, ok2 := isFac(lines[j]); ok2 {
					break
				}
				// Stop if a count-like line is followed by a faction line (start of next set)
				if v, ok3 := isSingleInt(lines[j]); ok3 && v > 0 {
					k := j + 1
					for k < len(lines) && strings.TrimSpace(lines[k]) == "" {
						k++
					}
					if k < len(lines) {
						if _, ok4 := isFac(lines[k]); ok4 {
							break
						}
					}
				}
				e.lines = append(e.lines, lines[j])
				j++
			}
			entries = append(entries, e)
		}

		if len(entries) != count {
			out = append(out, lines[i])
			for _, e := range entries {
				out = append(out, e.lines...)
			}
			i = j
			continue
		}

		hasDst, srcIdx := false, -1
		for k, e := range entries {
			if strings.EqualFold(e.name, dstFaction) {
				hasDst = true
			}
			if strings.EqualFold(e.name, srcFaction) {
				srcIdx = k
			}
		}
		newCount := count
		if !hasDst && srcIdx >= 0 {
			src := entries[srcIdx]
			dst := entry{
				name:  dstFaction,
				lines: make([]string, len(src.lines)),
			}
			copy(dst.lines, src.lines)
			orig := strings.TrimSpace(src.lines[0])
			trailing := src.lines[0][len(orig):]
			dst.lines[0] = fmt.Sprintf("%d %s", len(dstFaction), dstFaction) + trailing

			ins := make([]entry, 0, len(entries)+1)
			ins = append(ins, entries[:srcIdx+1]...)
			ins = append(ins, dst)
			ins = append(ins, entries[srcIdx+1:]...)
			entries = ins
			newCount++
			changed = true
		}

		out = append(out, strconv.Itoa(newCount))
		for _, e := range entries {
			out = append(out, e.lines...)
		}
		i = j
	}

	if !changed {
		return block, false
	}
	return strings.Join(out, "\n") + "\n", true
}

// DeleteBattleModelDraft removes a model entry by name from the draft modeldb.
// If the entry doesn't exist, it's a no-op.
func (w *GameWriter) DeleteBattleModelDraft(unitType string) error {
	if err := w.ensureInit(); err != nil {
		return err
	}

	relPath := w.modelDBRelPath()
	gameModelDB := filepath.Join(w.gamePath, relPath)
	if _, err := os.Stat(gameModelDB); err != nil {
		return nil // no modeldb at all
	}

	draftModelDB := filepath.Join(w.draftPath, relPath)

	loadPath := gameModelDB
	if _, err := os.Stat(draftModelDB); err == nil {
		loadPath = draftModelDB
	}

	db, err := parser.ParseBattleModels(loadPath)
	if err != nil {
		return fmt.Errorf("parse battle_models.modeldb: %w", err)
	}

	idx, exists := db.Index[unitType]
	if !exists {
		return nil // nothing to remove
	}

	newDB := &domain.BattleModelsDB{
		Header: db.Header,
		Count:  db.Count - 1,
		Models: make([]domain.BattleModel, 0, len(db.Models)-1),
		Index:  make(map[string]int, len(db.Index)-1),
	}
	for i, m := range db.Models {
		if i == idx {
			continue
		}
		newDB.Index[m.Name] = len(newDB.Models)
		newDB.Models = append(newDB.Models, m)
	}

	return WriteBattleModels(newDB, draftModelDB)
}
