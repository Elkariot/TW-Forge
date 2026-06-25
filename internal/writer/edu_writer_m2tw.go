package writer

import (
	"bufio"
	"fmt"
	"tw-forge/internal/domain"
	"tw-forge/internal/logger"
	"tw-forge/internal/repository"
	"os"
	"path/filepath"
	"strings"
)

func (w *GameWriter) SaveM2TWDraft(data domain.M2TWGameData, changes map[string]repository.ChangeType) error {
	if err := w.ensureInit(); err != nil {
		return err
	}
	if err := w.patchM2TWEDU(filepath.Join(w.draftPath, "export_descr_unit.txt"), data, changes); err != nil {
		return err
	}
	return w.SaveM2TWNamesDraft(data, changes)
}

func (w *GameWriter) patchM2TWEDU(dst string, data domain.M2TWGameData, changes map[string]repository.ChangeType) error {
	srcFile, err := os.Open(filepath.Join(w.backupPath, "export_descr_unit.txt"))
	if err != nil {
		return fmt.Errorf("open source M2TW EDU: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create draft M2TW EDU: %w", err)
	}
	defer dstFile.Close()

	unitIndex := make(map[string]domain.M2TWUnit, len(data.Units))
	for _, u := range data.Units {
		unitIndex[u.Type] = u
	}

	bw := bufio.NewWriter(dstFile)
	scanner := bufio.NewScanner(srcFile)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// M2TW файлы используют CRLF; Scanner.Text() снимает \r.
	writeLine := func(s string) { fmt.Fprintf(bw, "%s\r\n", s) }

	state := eduStateCopy

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		switch state {
		case eduStateCopy:
			unitType, isType := extractUnitType(trimmed)
			if !isType {
				writeLine(line)
				continue
			}
			changeType, changed := changes[unitType]
			if !changed {
				writeLine(line)
				continue
			}
			if changeType == repository.ChangeModified {
				for _, l := range serializeM2TWUnit(unitIndex[unitType]) {
					writeLine(l)
				}
			}
			state = eduStateSkip

		case eduStateSkip:
			if strings.HasPrefix(trimmed, "ownership") {
				state = eduStateSkipTail
			}

		case eduStateSkipTail:
			if trimmed == "" {
				writeLine(line)
				state = eduStateCopy
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan M2TW EDU: %w", err)
	}

	// Append added units
	for unitType, ct := range changes {
		if ct != repository.ChangeAdded {
			continue
		}
		u, ok := unitIndex[unitType]
		if !ok {
			continue
		}
		writeLine("")
		writeLine("")
		for _, l := range serializeM2TWUnit(u) {
			writeLine(l)
		}
	}

	if err := bw.Flush(); err != nil {
		return err
	}
	logger.FileWrite(dst, "write", "export_descr_unit.txt (M2TW)", 0)
	return nil
}
