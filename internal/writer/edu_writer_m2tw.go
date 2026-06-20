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

func (w *GameWriter) SaveM2TWDraft(data domain.M2TWGameData, changes map[string]repository.ChangeType) error {
	if err := w.ensureInit(); err != nil {
		return err
	}
	return w.patchM2TWEDU(filepath.Join(w.draftPath, "export_descr_unit.txt"), data, changes)
}

func (w *GameWriter) patchM2TWEDU(dst string, data domain.M2TWGameData, changes map[string]repository.ChangeType) error {
	srcFile, err := os.Open(filepath.Join(w.gamePath, "export_descr_unit.txt"))
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
				for _, l := range serializeM2TWUnit(unitIndex[unitType]) {
					fmt.Fprintln(bw, l)
				}
			}
			state = eduStateSkip

		case eduStateSkip:
			if strings.HasPrefix(trimmed, "ownership") {
				state = eduStateSkipTail
			}

		case eduStateSkipTail:
			if trimmed == "" {
				fmt.Fprintln(bw, line)
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
		fmt.Fprintln(bw, "")
		fmt.Fprintln(bw, "")
		for _, l := range serializeM2TWUnit(u) {
			fmt.Fprintln(bw, l)
		}
	}

	return bw.Flush()
}
