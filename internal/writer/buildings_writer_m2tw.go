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

func (w *GameWriter) SaveM2TWBuildingsDraft(data domain.M2TWGameData) error {
	if err := w.ensureInit(); err != nil {
		return err
	}
	dst := filepath.Join(w.draftPath, "export_descr_buildings.txt")
	return w.patchM2TWBuildings(dst, data)
}

func (w *GameWriter) patchM2TWBuildings(dst string, data domain.M2TWGameData) error {
	srcFile, err := os.Open(filepath.Join(w.backupPath, "export_descr_buildings.txt"))
	if err != nil {
		return fmt.Errorf("open M2TW buildings: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create draft M2TW buildings: %w", err)
	}
	defer dstFile.Close()

	type levelKey struct{ group, level string }
	levelIndex := make(map[levelKey]domain.M2TWBuildingLevel)
	for _, g := range data.Buildings {
		for _, l := range g.Levels {
			levelIndex[levelKey{g.Name, l.Name}] = l
		}
	}

	bw := bufio.NewWriter(dstFile)
	scanner := bufio.NewScanner(srcFile)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// M2TW data files use CRLF; Scanner.Text() strips \r.
	writeLine := func(s string) { fmt.Fprintf(bw, "%s\r\n", s) }

	depth := 0
	currentGroup := ""
	currentLevel := ""
	inCapBlock := false
	inUpgradeBlock := false
	inCapability := false
	inUpgrades := false
	capIndent := "\t\t\t\t"
	upgradeIndent := "\t\t\t\t"
	depWritten := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "{" {
			depth++
			if depth == 3 {
				depWritten = false
			}
			writeLine(line)
			if depth == 4 {
				if inCapability {
					inCapBlock = true
					inCapability = false
					k := levelKey{currentGroup, currentLevel}
					if lvl, ok := levelIndex[k]; ok {
						for _, bonus := range lvl.BonusLines {
							writeLine(capIndent + bonus)
						}
						for _, pool := range lvl.RecruitPools {
							writeLine(capIndent + formatM2TWRecruitPool(pool))
						}
					}
				} else if inUpgrades {
					inUpgradeBlock = true
					inUpgrades = false
					k := levelKey{currentGroup, currentLevel}
					if lvl, ok := levelIndex[k]; ok {
						for _, upg := range lvl.Upgrades {
							writeLine(upgradeIndent + upg)
						}
					}
				}
			}
			continue
		}

		if trimmed == "}" {
			if depth == 4 {
				inCapBlock = false
				inUpgradeBlock = false
			} else if depth == 3 && !depWritten {
				k := levelKey{currentGroup, currentLevel}
				if lvl, ok := levelIndex[k]; ok && lvl.Dependency != nil && lvl.Dependency.Group != "" && !lvl.DependencyInline {
					indent := leadingWhitespace(line) + "\t"
					writeLine(indent + "building_present_min_level " + lvl.Dependency.Group + " " + lvl.Dependency.Level)
				}
				depWritten = true
			}
			depth--
			writeLine(line)
			continue
		}

		if inCapBlock {
			if trimmed != "" && !strings.HasPrefix(trimmed, ";") {
				if lead := leadingWhitespace(line); lead != "" {
					capIndent = lead
				}
			}
			continue
		}
		if inUpgradeBlock {
			if trimmed != "" && !strings.HasPrefix(trimmed, ";") {
				if lead := leadingWhitespace(line); lead != "" {
					upgradeIndent = lead
				}
			}
			continue
		}

		if depth == 2 && trimmed != "" && !strings.HasPrefix(trimmed, ";") {
			fields := strings.Fields(trimmed)
			if len(fields) > 0 && fields[0] != "levels" {
				lvlName := fields[0]
				currentLevel = lvlName
				indent := leadingWhitespace(line)
				k := levelKey{currentGroup, lvlName}
				if lvl, ok := levelIndex[k]; ok {
					writeLine(indent + formatM2TWLevelDefinition(lvlName, lvl))
				} else {
					writeLine(line)
				}
				continue
			}
		}

		if depth == 3 && currentGroup != "" && currentLevel != "" && trimmed != "" && !strings.HasPrefix(trimmed, ";") {
			fields := strings.Fields(trimmed)
			indent := leadingWhitespace(line)
			k := levelKey{currentGroup, currentLevel}
			lvl, lvlOk := levelIndex[k]
			if len(fields) >= 1 && lvlOk {
				switch fields[0] {
				case "construction":
					writeLine(indent + "construction  " + strconv.Itoa(lvl.Construction))
					continue
				case "cost":
					writeLine(indent + "cost  " + strconv.Itoa(lvl.Cost))
					continue
				case "settlement_min":
					if lvl.SettlementMin != "" {
						writeLine(indent + "settlement_min " + lvl.SettlementMin)
					}
					continue
				case "convert_to":
					writeLine(indent + "convert_to " + strconv.Itoa(lvl.ConvertTo))
					continue
				case "building_present_min_level":
					continue // will inject updated version before capability/upgrades/}
				case "capability", "upgrades":
					if !depWritten {
						if lvl.Dependency != nil && lvl.Dependency.Group != "" && !lvl.DependencyInline {
							writeLine(indent + "building_present_min_level " + lvl.Dependency.Group + " " + lvl.Dependency.Level)
						}
						depWritten = true
					}
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
			switch fields[0] {
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
		return fmt.Errorf("scan M2TW buildings: %w", err)
	}
	return bw.Flush()
}

func formatM2TWLevelDefinition(name string, lvl domain.M2TWBuildingLevel) string {
	var sb strings.Builder
	sb.WriteString(name)
	if lvl.SettlementType != "" {
		sb.WriteString(" ")
		sb.WriteString(lvl.SettlementType)
	}
	if len(lvl.RequiredCultures) > 0 {
		sb.WriteString(" requires factions { ")
		for _, c := range lvl.RequiredCultures {
			sb.WriteString(c)
			sb.WriteString(", ")
		}
		sb.WriteString("}")
	}
	if lvl.DependencyInline && lvl.Dependency != nil && lvl.Dependency.Group != "" {
		sb.WriteString(" and building_present_min_level ")
		sb.WriteString(lvl.Dependency.Group)
		sb.WriteString(" ")
		sb.WriteString(lvl.Dependency.Level)
	}
	return sb.String()
}

func formatM2TWRecruitPool(pool domain.M2TWRecruitPool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `recruit_pool "%s"  %d   %.6g   %d  %d`,
		pool.UnitType, pool.InitialPool, pool.ReplenishRate, pool.MaxPool, pool.ExpGained)
	if len(pool.Factions) > 0 {
		sb.WriteString("  requires factions { ")
		for _, f := range pool.Factions {
			sb.WriteString(f)
			sb.WriteString(", ")
		}
		sb.WriteString("}")
	}
	if pool.Conditions != "" {
		sb.WriteString(" ")
		sb.WriteString(pool.Conditions)
	}
	return sb.String()
}
