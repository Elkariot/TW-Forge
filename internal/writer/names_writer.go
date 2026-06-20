package writer

import (
	"bufio"
	"fmt"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type namesScanState int

const (
	nsCopy         namesScanState = iota
	nsModSkipValue                // написали замену, пропускаем оригинальную строку значения
	nsDelSkip                     // удалённый юнит, пропускаем до следующего блока
)

// patchNames обновляет черновик export_units.txt (UTF-16 LE BOM).
//
// Формат файла:
//
//	{dictionary}\tИмя
//
//	{dictionary_descr}
//	Описание...
//
//	{dictionary_descr_short}
//	Короткое описание...
//
//	¬----------------
func (w *GameWriter) patchNames(dst string, data domain.GameData, changes map[string]repository.ChangeType) error {
	unitIndex := buildUnitIndex(data.Units)
	byDict := buildDictIndex(unitIndex, changes)

	srcFile, err := os.Open(filepath.Join(w.gamePath, "text", "export_units.txt"))
	if err != nil {
		return fmt.Errorf("open source export_units: %w", err)
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir for names draft: %w", err)
	}
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create draft names: %w", err)
	}
	defer dstFile.Close()

	codec := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)
	bw := bufio.NewWriter(transform.NewWriter(dstFile, codec.NewEncoder()))
	scanner := bufio.NewScanner(transform.NewReader(srcFile, codec.NewDecoder()))
	scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024)

	writeLine := func(s string) { fmt.Fprintf(bw, "%s\r\n", s) }

	state := nsCopy
	currentDict := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, "{") {
			// не тег: обычная строка, значение или комментарий
			switch state {
			case nsCopy:
				writeLine(line)
			case nsModSkipValue:
				if trimmed != "" {
					state = nsCopy // первая непустая строка — оригинальное значение, пропускаем
				} else {
					writeLine(line) // пустые строки сохраняем
				}
			case nsDelSkip:
				// пропускаем всё
			}
			continue
		}

		// тег: {tag} или {tag}\tvalue
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

		switch uc.changeType {
		case repository.ChangeDeleted:
			state = nsDelSkip

		case repository.ChangeModified:
			u := uc.unit
			switch field {
			case "name":
				writeLine(fmt.Sprintf("{%s}\t%s", u.Dictionary, u.Name))
				state = nsCopy // имя стоит inline — отдельной строки нет
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
		return fmt.Errorf("scan export_units: %w", err)
	}

	// Новые юниты дописываем в конец
	for unitType, ct := range changes {
		if ct != repository.ChangeAdded {
			continue
		}
		u, ok := unitIndex[unitType]
		if !ok {
			continue
		}
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

	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flush draft names: %w", err)
	}
	return nil
}

// parseDictKey извлекает базовый ключ словаря и имя поля из тега.
// "barb_peasant_briton_descr_short" → ("barb_peasant_briton", "descr_short")
func parseDictKey(tag string) (dictKey, field string) {
	if strings.HasSuffix(tag, "_descr_short") {
		return strings.TrimSuffix(tag, "_descr_short"), "descr_short"
	}
	if strings.HasSuffix(tag, "_descr") {
		return strings.TrimSuffix(tag, "_descr"), "descr"
	}
	return tag, "name"
}

type unitChange struct {
	unit       domain.Unit
	changeType repository.ChangeType
}

func buildUnitIndex(units []domain.Unit) map[string]domain.Unit {
	idx := make(map[string]domain.Unit, len(units))
	for _, u := range units {
		idx[u.Type] = u
	}
	return idx
}

func buildDictIndex(unitIndex map[string]domain.Unit, changes map[string]repository.ChangeType) map[string]unitChange {
	byDict := make(map[string]unitChange, len(changes))
	for unitType, ct := range changes {
		if u, ok := unitIndex[unitType]; ok {
			byDict[u.Dictionary] = unitChange{u, ct}
		}
	}
	return byDict
}
