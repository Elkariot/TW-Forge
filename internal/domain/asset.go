package domain

import "time"

// TemplateInfo describes one saved mod configuration snapshot (see writer.GameWriter's
// CreateTemplate/LoadTemplate).
type TemplateInfo struct {
	Name      string
	CreatedAt time.Time
}

// TemplateState reports which template (if any) the current game files were last
// captured from or switched to, and whether they've since drifted from it (edits made
// or applied after that point, not yet captured in any template).
type TemplateState struct {
	Current string
	Dirty   bool
}

// IconInfo — информация об иконке юнита.
type IconInfo struct {
	Data    string // base64 PNG или ""
	Source  string // "mod", "base", ""
	RelPath string // относительный путь от корня data, напр. "UI/units/julii/#foo.tga"
}

// AssetFile — один файл ресурса (модель или текстура).
type AssetFile struct {
	Name    string // только имя файла
	RelPath string // относительный путь от корня data
	Source  string // "mod" или "base"
}
