package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"tw-forge/internal/logger"
)

// CopyUnitModelAssets добавляет texture/model_sprite записи в descr_model_battle.txt
// и копирует обе иконки юнита: маленькую (#[TYPE].TGA) и портрет ([TYPE]_INFO.TGA).
func (w *GameWriter) CopyUnitModelAssets(soldierModel, srcUnitType, dstUnitType, srcFaction, dstFaction string) error {
	if soldierModel == "" || srcFaction == "" || dstFaction == "" {
		return nil
	}
	if err := w.ensureInit(); err != nil {
		return err
	}
	// Model/texture entries in descr_model_battle.txt are per-faction; no patch needed when faction is unchanged.
	if srcFaction != dstFaction {
		_ = w.patchDescBattleModels(soldierModel, srcFaction, dstFaction)
	}
	_ = w.copyUnitIcon(srcUnitType, dstUnitType, srcFaction, dstFaction)
	_ = w.copyUnitInfoCard(srcUnitType, dstUnitType, srcFaction, dstFaction)
	return nil
}

func (w *GameWriter) patchDescBattleModels(soldierModel, srcFaction, dstFaction string) error {
	origPath := filepath.Join(w.gamePath, "descr_model_battle.txt")
	if _, err := os.Stat(origPath); err != nil {
		return nil
	}

	backupPath := filepath.Join(w.backupPath, "descr_model_battle.txt")
	if _, err := os.Stat(backupPath); err != nil {
		_ = copyFile(origPath, backupPath)
	}

	draftPath := filepath.Join(w.draftPath, "descr_model_battle.txt")
	srcPath := origPath
	if _, err := os.Stat(draftPath); err == nil {
		srcPath = draftPath
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	patched, changed := addFactionModelEntries(string(data), soldierModel, srcFaction, dstFaction)
	if !changed {
		return nil
	}
	if err := os.WriteFile(draftPath, []byte(patched), 0644); err != nil {
		return err
	}
	logger.FileWrite(draftPath, "patch", "descr_model_battle.txt", 0)
	return nil
}

func addFactionModelEntries(content, soldierModel, srcFaction, dstFaction string) (string, bool) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	inTarget := false

	lastTextureLine := -1
	texturePath := ""
	texSep := "\t\t\t\t"
	hasDstTexture := false

	lastSpriteLine := -1
	spriteTail := ""
	spriteSep := "\t"
	hasDstSprite := false

	for i, raw := range lines {
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, ";") {
			continue
		}
		fields := strings.Fields(t)
		if len(fields) == 0 {
			continue
		}

		if fields[0] == "type" {
			if inTarget {
				break
			}
			if len(fields) >= 2 && fields[1] == soldierModel {
				inTarget = true
			}
			continue
		}

		if !inTarget {
			continue
		}

		if fields[0] == "texture" {
			rest := t[len("texture"):]
			if ci := strings.Index(rest, ","); ci != -1 {
				faction := strings.TrimSpace(rest[:ci])
				path := strings.TrimSpace(rest[ci+1:])
				if strings.EqualFold(faction, srcFaction) && texturePath == "" {
					texturePath = path
					rawAfter := raw[strings.Index(raw, "texture")+len("texture"):]
					if s := leadingWhitespace(rawAfter); s != "" {
						texSep = s
					}
				}
				if strings.EqualFold(faction, dstFaction) {
					hasDstTexture = true
				}
			}
			lastTextureLine = i
		}

		if fields[0] == "model_sprite" {
			rest := t[len("model_sprite"):]
			if ci := strings.Index(rest, ","); ci != -1 {
				faction := strings.TrimSpace(rest[:ci])
				tail := strings.TrimSpace(rest[ci+1:])
				if strings.EqualFold(faction, srcFaction) && spriteTail == "" {
					spriteTail = tail
					rawAfter := raw[strings.Index(raw, "model_sprite")+len("model_sprite"):]
					if s := leadingWhitespace(rawAfter); s != "" {
						spriteSep = s
					}
				}
				if strings.EqualFold(faction, dstFaction) {
					hasDstSprite = true
				}
			}
			lastSpriteLine = i
		}
	}

	if !inTarget {
		return content, false
	}

	type insertion struct {
		afterLine int
		newLine   string
	}
	var inserts []insertion

	if !hasDstTexture && texturePath != "" && lastTextureLine >= 0 {
		indent := leadingWhitespace(lines[lastTextureLine])
		inserts = append(inserts, insertion{
			lastTextureLine,
			fmt.Sprintf("%stexture%s%s, %s", indent, texSep, dstFaction, texturePath),
		})
	}
	if !hasDstSprite && spriteTail != "" && lastSpriteLine >= 0 {
		indent := leadingWhitespace(lines[lastSpriteLine])
		inserts = append(inserts, insertion{
			lastSpriteLine,
			fmt.Sprintf("%smodel_sprite%s%s, %s", indent, spriteSep, dstFaction, spriteTail),
		})
	}

	if len(inserts) == 0 {
		return content, false
	}

	sort.Slice(inserts, func(i, j int) bool {
		return inserts[i].afterLine > inserts[j].afterLine
	})
	for _, ins := range inserts {
		lines = sliceInsertAfter(lines, ins.afterLine, ins.newLine)
	}

	return strings.ReplaceAll(strings.Join(lines, "\n"), "\n", "\r\n"), true
}

func sliceInsertAfter(lines []string, idx int, newLine string) []string {
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:idx+1]...)
	out = append(out, newLine)
	out = append(out, lines[idx+1:]...)
	return out
}

// findUnitIconData ищет маленькую иконку юнита (#[TYPE].TGA) по типу юнита только на файловой системе.
func findUnitIconData(unitsDir, unitType, faction string) []byte {
	iconBase := "#" + strings.ReplaceAll(unitType, " ", "_")
	variants := []string{
		iconBase + ".tga",
		strings.ToLower(iconBase) + ".tga",
		strings.ToUpper(iconBase) + ".TGA",
	}

	tryDir := func(dir string) []byte {
		for _, name := range variants {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err == nil {
				return data
			}
		}
		return nil
	}

	if data := tryDir(filepath.Join(unitsDir, faction)); data != nil {
		return data
	}
	entries, _ := os.ReadDir(unitsDir)
	for _, e := range entries {
		if !e.IsDir() || strings.EqualFold(e.Name(), faction) {
			continue
		}
		if data := tryDir(filepath.Join(unitsDir, e.Name())); data != nil {
			return data
		}
	}
	return nil
}

// findUnitInfoData ищет портрет юнита ([TYPE]_INFO.TGA, 160×210) только на файловой системе.
// Паки не используются: их контент бывает некорректным (placeholder'ы или чужие портреты).
func findUnitInfoData(unitInfoDir, unitType, faction string) []byte {
	infoBase := strings.ReplaceAll(unitType, " ", "_") + "_info"
	variants := []string{
		infoBase + ".tga",
		strings.ToLower(infoBase) + ".tga",
		strings.ToUpper(infoBase) + ".TGA",
	}

	tryDir := func(dir string) []byte {
		for _, name := range variants {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err == nil {
				return data
			}
		}
		return nil
	}

	if data := tryDir(filepath.Join(unitInfoDir, faction)); data != nil {
		return data
	}
	entries, _ := os.ReadDir(unitInfoDir)
	for _, e := range entries {
		if !e.IsDir() || strings.EqualFold(e.Name(), faction) {
			continue
		}
		if data := tryDir(filepath.Join(unitInfoDir, e.Name())); data != nil {
			return data
		}
	}
	return nil
}

func (w *GameWriter) copyUnitIcon(srcUnitType, dstUnitType, srcFaction, dstFaction string) error {
	unitsDir := filepath.Join(w.gamePath, "UI", "units")
	iconData := findUnitIconData(unitsDir, srcUnitType, srcFaction)
	if iconData == nil {
		// Fallback: base RTW paks (never mod paks — they can contain wrong assets)
		if basePacksDir := findBasePacksDir(w.gamePath); basePacksDir != "" {
			iconName := "#" + strings.ToUpper(strings.ReplaceAll(srcUnitType, " ", "_")) + ".TGA"
			iconData = searchPaksDir(basePacksDir, iconName, strings.ToUpper(srcFaction))
		}
	}
	if iconData == nil {
		return nil
	}
	dstName := "#" + strings.ToUpper(strings.ReplaceAll(dstUnitType, " ", "_")) + ".TGA"
	dstDir := filepath.Join(w.draftPath, "UI", "units", dstFaction)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dstDir, dstName), iconData, 0644)
}

// copyUnitInfoCard копирует портрет юнита RTW ([TYPE]_INFO.TGA, 160×210) в папку новой фракции.
func (w *GameWriter) copyUnitInfoCard(srcUnitType, dstUnitType, srcFaction, dstFaction string) error {
	unitInfoDir := filepath.Join(w.gamePath, "UI", "unit_info")
	infoData := findUnitInfoData(unitInfoDir, srcUnitType, srcFaction)
	if infoData == nil {
		// Fallback: base RTW paks only
		if basePacksDir := findBasePacksDir(w.gamePath); basePacksDir != "" {
			infoName := strings.ToUpper(strings.ReplaceAll(srcUnitType, " ", "_")) + "_INFO.TGA"
			infoData = searchPaksDir(basePacksDir, infoName, strings.ToUpper(srcFaction))
		}
	}
	if infoData == nil {
		return nil
	}
	dstName := strings.ToUpper(strings.ReplaceAll(dstUnitType, " ", "_")) + "_INFO.TGA"
	dstDir := filepath.Join(w.draftPath, "UI", "unit_info", dstFaction)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dstDir, dstName), infoData, 0644)
}

// CopyUnitCardDraft копирует unit card иконку для M2TW:
// ui/units/[srcFaction]/#[srcUnitType].tga → draft/ui/units/[dstFaction]/#[dstUnitType].TGA
func (w *GameWriter) CopyUnitCardDraft(srcUnitType, dstUnitType, srcFaction, dstFaction string) error {
	if srcUnitType == "" || dstUnitType == "" || dstFaction == "" {
		return nil
	}
	if err := w.ensureInit(); err != nil {
		return err
	}
	unitsDir := filepath.Join(w.gamePath, "ui", "units")
	iconData := findUnitIconData(unitsDir, srcUnitType, srcFaction)
	if iconData == nil {
		return nil
	}
	dstName := "#" + strings.ToUpper(strings.ReplaceAll(dstUnitType, " ", "_")) + ".TGA"
	dstDir := filepath.Join(w.draftPath, "ui", "units", dstFaction)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dstDir, dstName), iconData, 0644)
}

// RenameUnitIcon переименовывает иконку RTW при смене типа юнита.
func (w *GameWriter) RenameUnitIcon(oldType, newType, faction string) error {
	return w.renameUnitIconDraft(oldType, newType, faction, "UI")
}

// RenameUnitCardDraft переименовывает иконку M2TW при смене типа юнита.
func (w *GameWriter) RenameUnitCardDraft(oldType, newType, faction string) error {
	return w.renameUnitIconDraft(oldType, newType, faction, "ui")
}

func (w *GameWriter) renameUnitIconDraft(oldType, newType, faction, unitsRoot string) error {
	if oldType == newType || faction == "" {
		return nil
	}
	if err := w.ensureInit(); err != nil {
		return err
	}

	unitsDir := filepath.Join(w.gamePath, unitsRoot, "units")
	iconData := findUnitIconData(unitsDir, oldType, faction)
	if iconData != nil {
		dstName := "#" + strings.ToUpper(strings.ReplaceAll(newType, " ", "_")) + ".TGA"
		dstDir := filepath.Join(w.draftPath, unitsRoot, "units", faction)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return err
		}

		file := filepath.Join(dstDir, dstName)
		_ = os.WriteFile(file, iconData, 0644)

		logger.FileWrite(file, "rename", file, 0)
	}
	// Для RTW также переименовываем портрет юнита (_info.tga)
	if strings.ToUpper(unitsRoot) == "UI" {
		unitInfoDir := filepath.Join(w.gamePath, unitsRoot, "unit_info")
		infoData := findUnitInfoData(unitInfoDir, oldType, faction)
		if infoData != nil {
			dstName := strings.ToUpper(strings.ReplaceAll(newType, " ", "_")) + "_INFO.TGA"
			dstDir := filepath.Join(w.draftPath, unitsRoot, "unit_info", faction)
			if err := os.MkdirAll(dstDir, 0755); err != nil {
				return err
			}
			_ = os.WriteFile(filepath.Join(dstDir, dstName), infoData, 0644)
		}
	}
	return nil
}

// DeleteUnitAssets удаляет иконки юнита (#[TYPE].TGA и [TYPE]_INFO.TGA) из всех
// папок фракций в game и draft директориях (для обоих RTW "UI" и M2TW "ui" вариантов).
func (w *GameWriter) DeleteUnitAssets(unitType string) error {
	if err := w.ensureInit(); err != nil {
		return err
	}
	typeKey := strings.ToUpper(strings.ReplaceAll(unitType, " ", "_"))
	smallName := "#" + typeKey + ".TGA"
	infoName := typeKey + "_INFO.TGA"

	for _, uiRoot := range []string{"UI", "ui"} {
		for _, root := range []string{w.gamePath, w.draftPath} {
			deleteIconFromSubdirs(filepath.Join(root, uiRoot, "units"), smallName)
			deleteIconFromSubdirs(filepath.Join(root, uiRoot, "unit_info"), infoName)
		}
	}
	return nil
}

func deleteIconFromSubdirs(dir, filename string) {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subDir := filepath.Join(dir, e.Name())
		_ = os.Remove(filepath.Join(subDir, filename))
		_ = os.Remove(filepath.Join(subDir, strings.ToLower(filename)))
	}
}

// applyDirRecursive копирует всё содержимое draft/[relDir]/ → game/[relDir]/.
func (w *GameWriter) applyDirRecursive(relDir string) error {
	draftDir := filepath.Join(w.draftPath, relDir)
	if _, err := os.Stat(draftDir); err != nil {
		return nil
	}
	return filepath.Walk(draftDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(draftDir, path)
		dst := filepath.Join(w.gamePath, relDir, rel)
		if mkErr := os.MkdirAll(filepath.Dir(dst), 0755); mkErr != nil {
			return mkErr
		}
		return copyFile(path, dst)
	})
}
