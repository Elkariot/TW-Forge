package domain

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
