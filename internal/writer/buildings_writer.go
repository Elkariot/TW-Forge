package writer

import (
	"bufio"
	"fmt"
	"modding-utils/internal/domain"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (w *GameWriter) SaveBuildingsDraft(data domain.GameData) error {
	if err := w.ensureInit(); err != nil {
		return err
	}
	dst := filepath.Join(w.draftPath, "export_descr_buildings.txt")
	return w.patchBuildings(dst, data)
}

func (w *GameWriter) patchBuildings(dst string, data domain.GameData) error {
	srcFile, err := os.Open(filepath.Join(w.gamePath, "export_descr_buildings.txt"))
	if err != nil {
		return fmt.Errorf("open buildings: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create draft buildings: %w", err)
	}
	defer dstFile.Close()

	// groupName+levelName → slots
	type key struct{ group, level string }
	slotIndex := make(map[key][]domain.RecruitSlot)
	for _, g := range data.Buildings {
		for _, l := range g.Levels {
			slotIndex[key{g.Name, l.Name}] = l.RecruitSlots
		}
	}

	bw := bufio.NewWriter(dstFile)
	scanner := bufio.NewScanner(srcFile)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	depth := 0
	currentGroup := ""
	currentLevel := ""
	inCapability := false  // "capability" keyword was seen at depth 3
	inCapBlock := false    // currently inside capability { } (depth 4)
	recruitIndent := "\t\t\t\t"

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "{" {
			depth++
			fmt.Fprintln(bw, line)
			if inCapability && depth == 4 {
				// Только что вошли в capability-блок — инжектируем актуальные recruit-строки
				inCapBlock = true
				inCapability = false
				slots := slotIndex[key{currentGroup, currentLevel}]
				for _, slot := range slots {
					fmt.Fprintln(bw, recruitIndent+formatBuildingSlot(slot))
				}
			}
			continue
		}

		if trimmed == "}" {
			if inCapBlock && depth == 4 {
				inCapBlock = false
			}
			depth--
			fmt.Fprintln(bw, line)
			continue
		}

		// Пропускаем оригинальные recruit-строки внутри capability-блока
		if inCapBlock && !strings.HasPrefix(trimmed, ";") && strings.HasPrefix(trimmed, "recruit ") {
			// Детектируем отступ из первой встреченной строки
			if lead := leadingWhitespace(line); lead != "" {
				recruitIndent = lead
			}
			continue
		}

		fmt.Fprintln(bw, line)

		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}

		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}

		switch depth {
		case 0:
			if fields[0] == "building" && len(fields) > 1 {
				currentGroup = fields[1]
			}
		case 2:
			// Строки вида "levelname requires factions {...}" или просто "levelname"
			if fields[0] != "levels" {
				currentLevel = fields[0]
			}
		case 3:
			if trimmed == "capability" {
				inCapability = true
				// Определяем отступ recruit-строк по отступу "capability" + один уровень
				capIndent := leadingWhitespace(line)
				if strings.Contains(capIndent, "\t") {
					recruitIndent = capIndent + "\t"
				} else if len(capIndent) > 0 {
					recruitIndent = capIndent + "    "
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan buildings: %w", err)
	}
	return bw.Flush()
}

func formatBuildingSlot(slot domain.RecruitSlot) string {
	var sb strings.Builder
	sb.WriteString("recruit \"")
	sb.WriteString(slot.UnitType)
	sb.WriteString("\" ")
	sb.WriteString(strconv.Itoa(slot.Level))
	if len(slot.Requirements) > 0 {
		sb.WriteString(" requires factions { ")
		for _, f := range slot.Requirements {
			sb.WriteString(f)
			sb.WriteString(", ")
		}
		sb.WriteString("}")
	}
	if slot.Conditions != "" {
		sb.WriteString(" ")
		sb.WriteString(slot.Conditions)
	}
	return sb.String()
}

func leadingWhitespace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
}
