package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CopyUnitModelAssets добавляет texture и model_sprite записи для dstFaction
// в descr_model_battle.txt и копирует иконку юнита в папку новой фракции.
// Best-effort: частичные сбои не прерывают операцию.
func (w *GameWriter) CopyUnitModelAssets(soldierModel, srcFaction, dstFaction string) error {
	if soldierModel == "" || srcFaction == "" || dstFaction == "" || srcFaction == dstFaction {
		return nil
	}
	if err := w.ensureInit(); err != nil {
		return err
	}
	_ = w.patchDescBattleModels(soldierModel, srcFaction, dstFaction)
	_ = w.copyUnitIcon(soldierModel, srcFaction, dstFaction)
	return nil
}

func (w *GameWriter) patchDescBattleModels(soldierModel, srcFaction, dstFaction string) error {
	origPath := filepath.Join(w.gamePath, "descr_model_battle.txt")
	if _, err := os.Stat(origPath); err != nil {
		return nil // RTW без этого файла — ничего не делаем
	}

	// Однократный бэкап оригинала
	backupPath := filepath.Join(w.backupPath, "descr_model_battle.txt")
	if _, err := os.Stat(backupPath); err != nil {
		_ = copyFile(origPath, backupPath)
	}

	// Читаем из черновика (если он уже есть от предыдущих операций), иначе из оригинала
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
	return os.WriteFile(draftPath, []byte(patched), 0644)
}

// addFactionModelEntries ищет блок `type soldierModel` и вставляет:
//   - `texture dstFaction, <path>` после последней texture-строки
//   - `model_sprite dstFaction, <range>, <path>` после последней model_sprite-строки
//
// Разделитель между ключом и значением копируется из существующих строк (не хардкодится).
// Если записи уже есть или блок не найден — unchanged=false.
func addFactionModelEntries(content, soldierModel, srcFaction, dstFaction string) (string, bool) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	inTarget := false

	// Texture
	lastTextureLine := -1
	texturePath := ""
	texSep := "\t\t\t\t" // fallback
	hasDstTexture := false

	// Model sprite
	lastSpriteLine := -1
	spriteTail := "" // "60.0, data/sprites/gauls_xxx.spr"
	spriteSep := "\t" // fallback
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
				break // вышли из блока целевой модели
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
			// Формат: texture   faction, path
			rest := t[len("texture"):]
			if ci := strings.Index(rest, ","); ci != -1 {
				faction := strings.TrimSpace(rest[:ci])
				path := strings.TrimSpace(rest[ci+1:])
				if strings.EqualFold(faction, srcFaction) && texturePath == "" {
					texturePath = path
					// Извлекаем реальный разделитель из необрезанной строки
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
			// Формат: model_sprite   faction, range, path
			rest := t[len("model_sprite"):]
			if ci := strings.Index(rest, ","); ci != -1 {
				faction := strings.TrimSpace(rest[:ci])
				tail := strings.TrimSpace(rest[ci+1:]) // "60.0, path"
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

	// Вставки от конца к началу, чтобы не сбивать индексы
	sort.Slice(inserts, func(i, j int) bool {
		return inserts[i].afterLine > inserts[j].afterLine
	})
	for _, ins := range inserts {
		lines = sliceInsertAfter(lines, ins.afterLine, ins.newLine)
	}

	// RTW требует CRLF
	return strings.ReplaceAll(strings.Join(lines, "\n"), "\n", "\r\n"), true
}

func sliceInsertAfter(lines []string, idx int, newLine string) []string {
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:idx+1]...)
	out = append(out, newLine)
	out = append(out, lines[idx+1:]...)
	return out
}

func (w *GameWriter) copyUnitIcon(soldierModel, srcFaction, dstFaction string) error {
	// RTW иконки: UI/units/[faction]/#[soldierModel].tga (регистр имени может быть разным)
	variants := []string{
		"#" + strings.ToLower(soldierModel) + ".tga",
		"#" + strings.ToUpper(soldierModel) + ".tga",
	}

	var srcIconPath, iconName string
	for _, name := range variants {
		p := filepath.Join(w.gamePath, "UI", "units", srcFaction, name)
		if _, err := os.Stat(p); err == nil {
			srcIconPath = p
			iconName = name
			break
		}
	}
	if srcIconPath == "" {
		return nil // иконки нет — best-effort
	}

	dstDir := filepath.Join(w.draftPath, "UI", "units", dstFaction)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	return copyFile(srcIconPath, filepath.Join(dstDir, iconName))
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
