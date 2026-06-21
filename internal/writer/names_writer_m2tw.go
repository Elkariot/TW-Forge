package writer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// SaveM2TWNamesDraft writes the export_units.txt draft for M2TW.
// If the mod has a loose export_units.txt — patches it.
// If not (vanilla/pack-based) — creates a minimal file with only changed/added units;
// the game will load vanilla names from packs and our entries will override/add on top.
func (w *GameWriter) SaveM2TWNamesDraft(data domain.M2TWGameData, changes map[string]repository.ChangeType) error {
	if err := w.ensureInit(); err != nil {
		return err
	}

	unitIndex := buildM2TWUnitIndex(data.Units)
	dstPath := filepath.Join(w.draftPath, "text", "export_units.txt")
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("mkdir for M2TW names draft: %w", err)
	}

	srcPath := filepath.Join(w.gamePath, "text", "export_units.txt")
	if _, err := os.Stat(srcPath); err != nil {
		// Нет loose-файла — создаём с нуля только с нашими изменениями.
		if err2 := w.createM2TWNamesFromScratch(dstPath, unitIndex, changes); err2 != nil {
			return err2
		}
	} else {
		if err2 := w.patchM2TWNames(srcPath, dstPath, unitIndex, changes); err2 != nil {
			return err2
		}
	}

	return w.writeM2TWEnums(unitIndex)
}

func (w *GameWriter) patchM2TWNames(srcPath, dstPath string, unitIndex map[string]domain.M2TWUnit, changes map[string]repository.ChangeType) error {
	byDict := buildM2TWDictIndex(unitIndex, changes)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source M2TW export_units: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create draft M2TW names: %w", err)
	}
	defer dstFile.Close()

	codec := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	bw := bufio.NewWriter(transform.NewWriter(dstFile, codec.NewEncoder()))
	scanner := bufio.NewScanner(transform.NewReader(srcFile, codec.NewDecoder()))
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	writeLine := func(s string) { fmt.Fprintf(bw, "%s\r\n", s) }

	state := nsCopy
	currentDict := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, "{") {
			switch state {
			case nsCopy:
				writeLine(line)
			case nsModSkipValue:
				if trimmed != "" {
					state = nsCopy
				} else {
					writeLine(line)
				}
			case nsDelSkip:
				// пропускаем
			}
			continue
		}

		end := strings.Index(trimmed, "}")
		if end == -1 {
			if state != nsDelSkip {
				writeLine(line)
			}
			continue
		}
		tag := trimmed[1:end]
		dictKey, field := parseDictKey(tag)

		if dictKey != currentDict {
			currentDict = dictKey
			state = nsCopy
		}

		uc, inChanges := byDict[dictKey]
		if !inChanges {
			writeLine(line)
			continue
		}

		switch uc.ct {
		case repository.ChangeDeleted:
			state = nsDelSkip
		case repository.ChangeModified:
			u := uc.unit
			switch field {
			case "name":
				writeLine(fmt.Sprintf("{%s}\t%s", u.Dictionary, u.Name))
				state = nsCopy
			case "descr":
				writeLine(fmt.Sprintf("{%s_descr}", u.Dictionary))
				writeLine(u.Descr)
				state = nsModSkipValue
			case "descr_short":
				writeLine(fmt.Sprintf("{%s_descr_short}", u.Dictionary))
				writeLine(u.DescrShort)
				state = nsModSkipValue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan M2TW export_units: %w", err)
	}

	for unitType, ct := range changes {
		if ct != repository.ChangeAdded {
			continue
		}
		if u, ok := unitIndex[unitType]; ok {
			writeM2TWUnitEntry(writeLine, u)
		}
	}

	return bw.Flush()
}

func (w *GameWriter) createM2TWNamesFromScratch(dstPath string, unitIndex map[string]domain.M2TWUnit, changes map[string]repository.ChangeType) error {
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create M2TW names draft: %w", err)
	}
	defer dstFile.Close()

	codec := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	bw := bufio.NewWriter(transform.NewWriter(dstFile, codec.NewEncoder()))
	writeLine := func(s string) { fmt.Fprintf(bw, "%s\r\n", s) }

	for unitType, ct := range changes {
		if ct == repository.ChangeDeleted {
			continue
		}
		if u, ok := unitIndex[unitType]; ok {
			writeM2TWUnitEntry(writeLine, u)
		}
	}

	return bw.Flush()
}

// writeM2TWEnums пересоздаёт export_descr_unit_enums.txt из ВСЕХ юнитов.
// Если файла нет в папке игры — не создаём (мод не использует loose enums).
// Если файл есть — генерируем заново: все существующие юниты + новые.
// Это безопасно: .strings.bin кэш устаревает и M2TW пересоздаёт его при следующем запуске.
func (w *GameWriter) writeM2TWEnums(unitIndex map[string]domain.M2TWUnit) error {
	srcPath := filepath.Join(w.gamePath, "export_descr_unit_enums.txt")
	if _, err := os.Stat(srcPath); err != nil {
		return nil // мод без loose enums — не трогаем
	}

	dstPath := filepath.Join(w.draftPath, "export_descr_unit_enums.txt")

	var sb strings.Builder
	for _, u := range unitIndex {
		if u.IsDeleted {
			continue
		}
		sb.WriteString(u.Dictionary)
		sb.WriteByte('\n')
		sb.WriteString(u.Dictionary)
		sb.WriteString("_descr\n")
		sb.WriteString(u.Dictionary)
		sb.WriteString("_descr_short\n")
	}

	return os.WriteFile(dstPath, []byte(sb.String()), 0644)
}

func writeM2TWUnitEntry(writeLine func(string), u domain.M2TWUnit) {
	writeLine("")
	writeLine(fmt.Sprintf("{%s}\t%s", u.Dictionary, u.Name))
	writeLine("")
	writeLine(fmt.Sprintf("{%s_descr}", u.Dictionary))
	writeLine(u.Descr)
	writeLine("")
	writeLine(fmt.Sprintf("{%s_descr_short}", u.Dictionary))
	writeLine(u.DescrShort)
	writeLine("")
	writeLine("¬----------------")
	writeLine("")
}

type m2twUnitChange struct {
	unit domain.M2TWUnit
	ct   repository.ChangeType
}

func buildM2TWUnitIndex(units []domain.M2TWUnit) map[string]domain.M2TWUnit {
	idx := make(map[string]domain.M2TWUnit, len(units))
	for _, u := range units {
		idx[u.Type] = u
	}
	return idx
}

func buildM2TWDictIndex(unitIndex map[string]domain.M2TWUnit, changes map[string]repository.ChangeType) map[string]m2twUnitChange {
	byDict := make(map[string]m2twUnitChange, len(changes))
	for unitType, ct := range changes {
		if u, ok := unitIndex[unitType]; ok {
			byDict[u.Dictionary] = m2twUnitChange{u, ct}
		}
	}
	return byDict
}
