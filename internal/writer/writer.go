package writer

import (
	"fmt"
	"io"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"
	"os"
	"path/filepath"
	"sync"
)

var managedFiles = []string{
	"export_descr_unit.txt",
	"export_descr_unit_enums.txt",
	"export_descr_buildings.txt",
	filepath.Join("text", "export_units.txt"),
}

type GameWriter struct {
	gamePath   string
	backupPath string
	draftPath  string
	initOnce   sync.Once
	initErr    error
}

func New(gamePath string) *GameWriter {
	editorRoot := filepath.Join(filepath.Dir(gamePath), "_modding_editor")
	return &GameWriter{
		gamePath:   gamePath,
		backupPath: filepath.Join(editorRoot, "backup"),
		draftPath:  filepath.Join(editorRoot, "draft"),
	}
}

func (w *GameWriter) SaveDraft(data domain.GameData, changes map[string]repository.ChangeType) error {
	if err := w.ensureInit(); err != nil {
		return err
	}
	if err := w.patchEDU(filepath.Join(w.draftPath, "export_descr_unit.txt"), data, changes); err != nil {
		return err
	}
	return w.patchNames(filepath.Join(w.draftPath, "text", "export_units.txt"), data, changes)
}

func (w *GameWriter) Apply() error {
	if err := w.ensureInit(); err != nil {
		return err
	}
	if err := w.applyFile("export_descr_unit.txt"); err != nil {
		return err
	}
	draftNames := filepath.Join(w.draftPath, "text", "export_units.txt")
	if _, err := os.Stat(draftNames); err == nil {
		if err := w.applyFile(filepath.Join("text", "export_units.txt")); err != nil {
			return err
		}
	}
	draftBuildings := filepath.Join(w.draftPath, "export_descr_buildings.txt")
	if _, err := os.Stat(draftBuildings); err == nil {
		if err := w.applyFile("export_descr_buildings.txt"); err != nil {
			return err
		}
	}
	draftModelDB := filepath.Join(w.draftPath, "battle_models.modeldb")
	if _, err := os.Stat(draftModelDB); err == nil {
		if err := w.applyFile("battle_models.modeldb"); err != nil {
			return err
		}
	}
	return nil
}

func (w *GameWriter) ensureInit() error {
	w.initOnce.Do(func() {
		w.initErr = w.init()
	})
	return w.initErr
}

func (w *GameWriter) init() error {
	if err := os.MkdirAll(w.backupPath, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}
	if err := os.MkdirAll(w.draftPath, 0755); err != nil {
		return fmt.Errorf("create draft dir: %w", err)
	}
	return w.backupOriginals()
}

func (w *GameWriter) backupOriginals() error {
	for _, filename := range managedFiles {
		dst := filepath.Join(w.backupPath, filename)
		if _, err := os.Stat(dst); err == nil {
			continue // уже есть
		}
		src := filepath.Join(w.gamePath, filename)
		if _, err := os.Stat(src); err != nil {
			continue // файл не существует в данной версии игры
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("mkdir backup dir for %s: %w", filename, err)
		}
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("backup %s: %w", filename, err)
		}
	}
	return nil
}

func (w *GameWriter) applyFile(filename string) error {
	src := filepath.Join(w.draftPath, filename)
	dst := filepath.Join(w.gamePath, filename)
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
