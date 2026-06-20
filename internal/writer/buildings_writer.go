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

	depth := 0
	currentGroup := ""
	currentLevel := ""

	inCapability := false
	inCapBlock := false
	capIndent := "\t\t\t\t"

	inUpgrades := false
	inUpgradeBlock := false
	upgradeIndent := "\t\t\t\t"

	depWritten := false // tracking whether we wrote building_present_min_level for current level

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "{" {
			depth++
			if depth == 3 {
				depWritten = false
			}
			fmt.Fprintln(bw, line)

			if depth == 4 {
				if inCapability {
					inCapBlock = true
					inCapability = false
					k := key{currentGroup, currentLevel}
					lvl := levelIndex[k]
					for _, bonus := range lvl.BonusLines {
						fmt.Fprintln(bw, capIndent+bonus)
					}
					for _, slot := range lvl.RecruitSlots {
						fmt.Fprintln(bw, capIndent+formatBuildingSlot(slot))
					}
				} else if inUpgrades {
					inUpgradeBlock = true
					inUpgrades = false
					k := key{currentGroup, currentLevel}
					for _, upg := range levelIndex[k].Upgrades {
						fmt.Fprintln(bw, upgradeIndent+upg)
					}
				}
			}
			continue
		}

		if trimmed == "}" {
			if depth == 4 {
				if inCapBlock {
					inCapBlock = false
				} else if inUpgradeBlock {
					inUpgradeBlock = false
				}
			} else if depth == 3 && !depWritten && currentGroup != "" && currentLevel != "" {
				// Inject dependency before closing level block if not yet written
				k := key{currentGroup, currentLevel}
				if lvl, ok := levelIndex[k]; ok && lvl.Dependency != nil && lvl.Dependency.Group != "" {
					indent := leadingWhitespace(line) + "\t"
					fmt.Fprintln(bw, indent+"building_present_min_level "+lvl.Dependency.Group+" "+lvl.Dependency.Level)
				}
				depWritten = true
			}
			depth--
			fmt.Fprintln(bw, line)
			continue
		}

		// Пропускаем всё содержимое capability-блока и upgrades-блока.
		if trimmed != "" && !strings.HasPrefix(trimmed, ";") {
			if inCapBlock {
				if lead := leadingWhitespace(line); lead != "" {
					capIndent = lead
				}
				continue
			}
			if inUpgradeBlock {
				if lead := leadingWhitespace(line); lead != "" {
					upgradeIndent = lead
				}
				continue
			}
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
					fmt.Fprintln(bw, indent+formatLevelDefinition(lvlName, lvl))
				} else {
					fmt.Fprintln(bw, line)
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
					fmt.Fprintln(bw, indent+"construction\t"+strconv.Itoa(lvl.Construction))
					continue
				case "cost":
					fmt.Fprintln(bw, indent+"cost\t"+strconv.Itoa(lvl.Cost))
					continue
				case "settlement_min":
					if lvl.SettlementMin != "" {
						fmt.Fprintln(bw, indent+"settlement_min "+lvl.SettlementMin)
					}
					continue
				case "building_present_min_level":
					// Always skip original; will inject updated value before capability/upgrades/}
					continue
				case "capability", "upgrades":
					if !depWritten {
						if lvl.Dependency != nil && lvl.Dependency.Group != "" {
							fmt.Fprintln(bw, indent+"building_present_min_level "+lvl.Dependency.Group+" "+lvl.Dependency.Level)
						}
						depWritten = true
					}
				}
			}
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
