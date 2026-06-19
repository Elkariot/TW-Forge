package writer

import (
	"bufio"
	"fmt"
	"modding-utils/internal/domain"
	"modding-utils/internal/repository"
	"os"
	"path/filepath"
	"strings"
)

type eduScanState int

const (
	eduStateCopy     eduScanState = iota
	eduStateSkip                  // внутри изменённого блока, пропускаем оригинал
	eduStateSkipTail              // увидели ownership, ждём первую пустую строку
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

	state := eduStateCopy

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		switch state {
		case eduStateCopy:
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
			state = eduStateSkip

		case eduStateSkip:
			if strings.HasPrefix(trimmed, "ownership") {
				state = eduStateSkipTail
			}

		case eduStateSkipTail:
			if trimmed == "" {
				fmt.Fprintln(bw, line) // сохраняем пустую строку-разделитель
				state = eduStateCopy
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan EDU: %w", err)
	}

	// Добавленные юниты дописываем в конец
	for unitType, ct := range changes {
		if ct != repository.ChangeAdded {
			continue
		}
		u, ok := unitIndex[unitType]
		if !ok {
			continue
		}
		fmt.Fprintln(bw, "")
		fmt.Fprintln(bw, "")
		for _, l := range serializeUnit(u) {
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
