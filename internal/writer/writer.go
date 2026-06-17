package writer

import (
	"bufio"
	"fmt"
	"io"
	"modding-utils/internal/domain"
	"modding-utils/internal/repository"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Файлы, которыми управляет редактор — копируются в backup при первом запуске.
var managedFiles = []string{
	"export_descr_unit.txt",
	"export_descr_unit_enums.txt",
	"export_descr_buildings.txt",
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
	dst := filepath.Join(w.draftPath, "export_descr_unit.txt")
	return w.patchEDU(dst, data, changes)
}

func (w *GameWriter) Apply() error {
	if err := w.ensureInit(); err != nil {
		return err
	}
	return w.applyFile("export_descr_unit.txt")
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

type scanState int

const (
	stateCopy     scanState = iota
	stateSkip               // внутри изменённого блока, пропускаем оригинал
	stateSkipTail           // увидели ownership, ждём первую пустую строку
)

func (w *GameWriter) patchEDU(dst string, data domain.GameData, changes map[string]repository.ChangeType) error {
	srcFile, err := os.Open(filepath.Join(w.gamePath, "export_descr_unit.txt"))
	if err != nil {
		return fmt.Errorf("open source EDU: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create draft EDU: %w", err)
	}
	defer dstFile.Close()

	unitIndex := make(map[string]domain.Unit, len(data.Units))
	for _, u := range data.Units {
		unitIndex[u.Type] = u
	}

	bw := bufio.NewWriter(dstFile)
	scanner := bufio.NewScanner(srcFile)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	state := stateCopy

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		switch state {
		case stateCopy:
			unitType, isType := extractUnitType(trimmed)
			if !isType {
				fmt.Fprintln(bw, line)
				continue
			}

			changeType, changed := changes[unitType]
			if !changed {
				fmt.Fprintln(bw, line)
				continue
			}

			if changeType == repository.ChangeModified {
				for _, l := range serializeUnit(unitIndex[unitType]) {
					fmt.Fprintln(bw, l)
				}
			}
			// ChangeDeleted: ничего не пишем
			state = stateSkip

		case stateSkip:
			if strings.HasPrefix(trimmed, "ownership") {
				state = stateSkipTail
			}

		case stateSkipTail:
			if trimmed == "" {
				fmt.Fprintln(bw, line) // сохраняем пустую строку-разделитель
				state = stateCopy
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan EDU: %w", err)
	}

	// Добавленные юниты дописываем в конец
	for unitType, changeType := range changes {
		if changeType != repository.ChangeAdded {
			continue
		}
		unit, ok := unitIndex[unitType]
		if !ok {
			continue
		}
		fmt.Fprintln(bw, "")
		fmt.Fprintln(bw, "")
		for _, l := range serializeUnit(unit) {
			fmt.Fprintln(bw, l)
		}
	}

	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flush draft EDU: %w", err)
	}
	return nil
}

// extractUnitType возвращает тип юнита если строка является объявлением "type ...".
func extractUnitType(trimmedLine string) (string, bool) {
	if i := strings.Index(trimmedLine, ";"); i != -1 {
		trimmedLine = strings.TrimSpace(trimmedLine[:i])
	}
	fields := strings.Fields(trimmedLine)
	if len(fields) < 2 || fields[0] != "type" {
		return "", false
	}
	return strings.Join(fields[1:], " "), true
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
