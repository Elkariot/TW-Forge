package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CopyUnitModelAssets добавляет texture-запись для dstFaction в descr_model_battle.txt
// и копирует иконку юнита в папку новой фракции.
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

	patched, changed := addFactionTextureLine(string(data), soldierModel, srcFaction, dstFaction)
	if !changed {
		return nil
	}
	return os.WriteFile(draftPath, []byte(patched), 0644)
}

// addFactionTextureLine ищет блок `type soldierModel` и вставляет строку
// `texture dstFaction, <path>` после последней существующей texture-строки.
// Если запись для dstFaction уже есть или блок/путь не найден — unchanged=false.
func addFactionTextureLine(content, soldierModel, srcFaction, dstFaction string) (string, bool) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	inTarget := false
	lastTextureLine := -1
	texturePath := ""
	hasDst := false

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
			// Формат: texture   faction, path/to/texture.tga
			rest := t[len("texture"):]
			commaIdx := strings.Index(rest, ",")
			if commaIdx != -1 {
				faction := strings.TrimSpace(rest[:commaIdx])
				path := strings.TrimSpace(rest[commaIdx+1:])
				if strings.EqualFold(faction, srcFaction) && texturePath == "" {
					texturePath = path
				}
				if strings.EqualFold(faction, dstFaction) {
					hasDst = true
				}
			}
			lastTextureLine = i
		}
	}

	if !inTarget || hasDst || texturePath == "" || lastTextureLine < 0 {
		return content, false
	}

	// Сохраняем ведущие пробелы/табы исходной texture-строки
	indent := leadingWhitespace(lines[lastTextureLine])
	newLine := fmt.Sprintf("%stexture\t\t\t\t%s, %s", indent, dstFaction, texturePath)

	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:lastTextureLine+1]...)
	out = append(out, newLine)
	out = append(out, lines[lastTextureLine+1:]...)

	// RTW требует CRLF
	return strings.ReplaceAll(strings.Join(out, "\n"), "\n", "\r\n"), true
}

func (w *GameWriter) copyUnitIcon(soldierModel, srcFaction, dstFaction string) error {
	// RTW иконки: UI/units/[faction]/#[soldierModel].tga (имя может быть в любом регистре)
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
