package writer

import (
	"bufio"
	"fmt"
	"tw-forge/internal/domain"
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

	type key struct{ group, level string }
	levelIndex := make(map[key]domain.BuildingLevel)
	for _, g := range data.Buildings {
		for _, l := range g.Levels {
			levelIndex[key{g.Name, l.Name}] = l
		}
	}

	bw := bufio.NewWriter(dstFile)
	scanner := bufio.NewScanner(srcFile)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// RTW ожидает CRLF; Scanner.Text() снимает \r.
	writeLine := func(s string) { fmt.Fprintf(bw, "%s\r\n", s) }

	depth := 0
	currentGroup := ""
	currentLevel := ""

	inCapability := false
	inCapBlock := false
	capIndent := "\t\t\t\t"
	writtenCapSlots := map[string]bool{} // unitType → уже записан из оригинала

	inUpgrades := false
	inUpgradeBlock := false
	upgradeIndent := "\t\t\t\t"

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "{" {
			depth++
			writeLine(line)

			if depth == 4 {
				if inCapability {
					inCapBlock = true
					inCapability = false
					writtenCapSlots = map[string]bool{}
				} else if inUpgrades {
					inUpgradeBlock = true
					inUpgrades = false
					k := key{currentGroup, currentLevel}
					for _, upg := range levelIndex[k].Upgrades {
						writeLine(upgradeIndent + upg)
					}
				}
			}
			continue
		}

		if trimmed == "}" {
			if depth == 4 {
				if inCapBlock {
					// Дописываем новые слоты, которых не было в оригинале
					k := key{currentGroup, currentLevel}
					for _, slot := range levelIndex[k].RecruitSlots {
						if !writtenCapSlots[slot.UnitType] {
							writeLine(capIndent + formatBuildingSlot(slot))
						}
					}
					inCapBlock = false
				} else if inUpgradeBlock {
					inUpgradeBlock = false
				}
			}
			depth--
			writeLine(line)
			continue
		}

		// Обрабатываем содержимое capability-блока построчно.
		if inCapBlock {
			if lead := leadingWhitespace(line); lead != "" {
				capIndent = lead
			}
			if trimmed == "" || strings.HasPrefix(trimmed, ";") {
				writeLine(line)
				continue
			}
			fields := strings.Fields(trimmed)
			if len(fields) > 0 && fields[0] == "recruit" {
				// Обновляем или удаляем слот вербовки
				unitType := extractRecruitType(trimmed)
				k := key{currentGroup, currentLevel}
				found := false
				for _, slot := range levelIndex[k].RecruitSlots {
					if slot.UnitType == unitType {
						writeLine(capIndent + formatBuildingSlot(slot))
						writtenCapSlots[unitType] = true
						found = true
						break
					}
				}
				_ = found // если не найден — слот удалён, пропускаем
			} else {
				// Не recruit-строка (law_bonus, happiness_bonus и т.п.) — сохраняем как есть
				writeLine(line)
			}
			continue
		}

		// Пропускаем содержимое upgrades-блока (переписываем из памяти).
		if inUpgradeBlock {
			if trimmed != "" && !strings.HasPrefix(trimmed, ";") {
				if lead := leadingWhitespace(line); lead != "" {
					upgradeIndent = lead
				}
			}
			continue
		}

		// Заменяем строку определения уровня (depth 2).
		if depth == 2 && trimmed != "" && !strings.HasPrefix(trimmed, ";") {
			fields := strings.Fields(trimmed)
			if len(fields) > 0 && fields[0] != "levels" {
				lvlName := fields[0]
				currentLevel = lvlName
				indent := leadingWhitespace(line)
				k := key{currentGroup, lvlName}
				if lvl, ok := levelIndex[k]; ok {
					writeLine(indent + formatLevelDefinition(lvlName, lvl))
				} else {
					writeLine(line)
				}
				continue
			}
		}

		// Заменяем свойства уровня здания (depth 3).
		if depth == 3 && currentGroup != "" && currentLevel != "" && trimmed != "" && !strings.HasPrefix(trimmed, ";") {
			fields := strings.Fields(trimmed)
			indent := leadingWhitespace(line)
			k := key{currentGroup, currentLevel}
			lvl := levelIndex[k]
			if len(fields) >= 1 {
				switch fields[0] {
				case "construction":
					writeLine(indent + "construction\t" + strconv.Itoa(lvl.Construction))
					continue
				case "cost":
					writeLine(indent + "cost\t" + strconv.Itoa(lvl.Cost))
					continue
				case "settlement_min":
					if lvl.SettlementMin != "" {
						writeLine(indent + "settlement_min " + lvl.SettlementMin)
					}
					continue
				case "building_present_min_level":
					// Skip: dependency is now written inline on the level definition line.
					continue
				}
			}
		}

		writeLine(line)

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
		case 3:
			switch trimmed {
			case "capability":
				inCapability = true
				ci := leadingWhitespace(line)
				if strings.Contains(ci, "\t") {
					capIndent = ci + "\t"
				} else if len(ci) > 0 {
					capIndent = ci + "    "
				}
			case "upgrades":
				inUpgrades = true
				ui := leadingWhitespace(line)
				if strings.Contains(ui, "\t") {
					upgradeIndent = ui + "\t"
				} else if len(ui) > 0 {
					upgradeIndent = ui + "    "
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan buildings: %w", err)
	}
	return bw.Flush()
}

func formatLevelDefinition(name string, lvl domain.BuildingLevel) string {
	var sb strings.Builder
	sb.WriteString(name)
	if len(lvl.RequiredCultures) > 0 {
		sb.WriteString(" requires factions { ")
		for _, c := range lvl.RequiredCultures {
			sb.WriteString(c)
			sb.WriteString(", ")
		}
		sb.WriteString("}")
	}
	if lvl.Dependency != nil && lvl.Dependency.Group != "" {
		sb.WriteString(" and building_present_min_level ")
		sb.WriteString(lvl.Dependency.Group)
		sb.WriteString(" ")
		sb.WriteString(lvl.Dependency.Level)
	}
	return sb.String()
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

// extractRecruitType извлекает тип юнита из строки recruit "unit_type" ...
func extractRecruitType(line string) string {
	start := strings.Index(line, `"`)
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start+1:], `"`)
	if end == -1 {
		return ""
	}
	return line[start+1 : start+1+end]
}
