package writer

import (
	"fmt"
	"modding-utils/internal/parser"
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

	gameModelDB := filepath.Join(w.gamePath, "battle_models.modeldb")
	draftModelDB := filepath.Join(w.draftPath, "battle_models.modeldb")

	// If the source file doesn't exist at all, skip silently.
	if _, err := os.Stat(gameModelDB); err != nil {
		return fmt.Errorf("battle_models.modeldb not found in game directory")
	}

	// Back up original once.
	backupModelDB := filepath.Join(w.backupPath, "battle_models.modeldb")
	if _, err := os.Stat(backupModelDB); err != nil {
		if err := copyFile(gameModelDB, backupModelDB); err != nil {
			return fmt.Errorf("backup battle_models.modeldb: %w", err)
		}
	}

	// Load from draft (already modified) or original.
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
