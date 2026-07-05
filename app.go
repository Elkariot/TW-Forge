package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"tw-forge/internal/config"
	"tw-forge/internal/domain"
	"tw-forge/internal/logger"
	"tw-forge/internal/parser"
	"tw-forge/internal/repository"
	"tw-forge/internal/service"
	"tw-forge/internal/tgadecoder"
	"tw-forge/internal/writer"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type AppConfig struct {
	Game int `json:"game"`
	// Per-game data (key = strconv.Itoa(config.GameVersion))
	GamePaths        map[string]string            `json:"gamePaths,omitempty"`
	SelectedModPaths map[string]string            `json:"selectedModPaths,omitempty"`
	ManualMods       map[string]map[string]string `json:"manualMods,omitempty"`
	// Legacy single-game fields (for backward compat with old config files)
	LegacyGamePath string            `json:"gamePath,omitempty"`
	LegacySelMod   string            `json:"selectedModPath,omitempty"`
	LegacyMods     map[string]string `json:"legacyMods,omitempty"`
}

func appConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(dir, "total-war-mod-editor")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "config.json"), nil
}

func (a *App) LoadAppConfig() AppConfig {
	path, err := appConfigPath()
	if err != nil {
		return AppConfig{Game: -1}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{Game: -1}
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{Game: -1}
	}
	return cfg
}

func (a *App) SaveAppConfig(game int, gamePath, selectedModPath string) error {
	cfgPath, err := appConfigPath()
	if err != nil {
		return err
	}
	cfg := a.loadRawConfig()
	cfg.Game = game
	key := fmt.Sprintf("%d", game)
	if cfg.GamePaths == nil {
		cfg.GamePaths = make(map[string]string)
	}
	if cfg.SelectedModPaths == nil {
		cfg.SelectedModPaths = make(map[string]string)
	}
	cfg.GamePaths[key] = gamePath
	cfg.SelectedModPaths[key] = selectedModPath
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}

func (a *App) loadRawConfig() AppConfig {
	path, err := appConfigPath()
	if err != nil {
		return AppConfig{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}
	}
	var cfg AppConfig
	_ = json.Unmarshal(data, &cfg)
	// Migrate legacy single-game fields into per-game maps.
	if cfg.LegacyGamePath != "" && len(cfg.GamePaths) == 0 {
		key := fmt.Sprintf("%d", cfg.Game)
		cfg.GamePaths = map[string]string{key: cfg.LegacyGamePath}
		cfg.SelectedModPaths = map[string]string{key: cfg.LegacySelMod}
		if len(cfg.LegacyMods) > 0 {
			cfg.ManualMods = map[string]map[string]string{key: cfg.LegacyMods}
		}
	}
	return cfg
}

type App struct {
	ctx              context.Context
	gamePath         string
	baseGameDataPath string // пустая если мод не выбран (gamePath и есть база)
	gameVersion      config.GameVersion
	generalService   *service.GeneralService
	// RTW
	unitService     *service.UnitService
	factionService  *service.FactionService
	buildingService *service.BuildingService
	rtwRepo         repository.GameRepository
	// M2TW (non-nil only when gameVersion == config.Medieval)
	m2twUnitService     *service.M2TWUnitService
	m2twBuildingService *service.M2TWBuildingService
	m2twRepo            repository.M2TWGameRepository
	// Shared between RTW and M2TW: descr_projectile.txt format is identical.
	projectileService *service.ProjectileService
	projectileRepo    repository.ProjectileRepository
	// Shared writer (non-nil after InitGame)
	gameWriter *writer.GameWriter
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.Init(logger.DefaultPath())
	logger.Info("tw-forge started")
	a.generalService = service.NewGeneralService(nil)
}

// OpenDirectoryDialog открывает нативный диалог выбора папки.
func (a *App) OpenDirectoryDialog() string {
	path, _ := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Выберите папку с игрой",
	})
	return path
}

// InitGameFolder валидирует корневую папку игры и сканирует моды.
func (a *App) InitGameFolder(game int, path string) error {
	if err := a.generalService.InitGameFolder(config.GameVersion(game), path); err != nil {
		return err
	}
	_ = a.generalService.CheckModsFolder()
	return nil
}

// GetMods возвращает найденные моды: name → dataPath.
func (a *App) GetMods() map[string]string {
	return a.generalService.GetMods()
}

// GetBaseGameDataPath возвращает путь к data/ основной игры.
func (a *App) GetBaseGameDataPath() string {
	return a.generalService.GetGameSettings().GamePath
}

// AddModPath добавляет папку мода вручную и сохраняет в конфиг (per-game).
func (a *App) AddModPath(game int, modPath, name string) error {
	if err := a.generalService.AddModPath(modPath, name); err != nil {
		return err
	}
	cfgPath, err := appConfigPath()
	if err != nil {
		return nil // не критично
	}
	cfg := a.loadRawConfig()
	key := fmt.Sprintf("%d", game)
	if cfg.ManualMods == nil {
		cfg.ManualMods = make(map[string]map[string]string)
	}
	if cfg.ManualMods[key] == nil {
		cfg.ManualMods[key] = make(map[string]string)
	}
	cfg.ManualMods[key][name] = modPath
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil
	}
	_ = os.WriteFile(cfgPath, data, 0644)
	return nil
}

// InitGame вызывается с фронтенда когда пользователь выбрал путь к игре.
func (a *App) InitGame(gamePath string, gameVersion config.GameVersion) error {
	a.gamePath = gamePath
	a.gameVersion = gameVersion

	// Определяем базовый путь к данным игры из конфига.
	cfg := a.loadRawConfig()
	key := fmt.Sprintf("%d", int(gameVersion))
	if baseGamePath := cfg.GamePaths[key]; baseGamePath != "" {
		baseData := filepath.Join(baseGamePath, "data")
		if !strings.EqualFold(filepath.Clean(baseData), filepath.Clean(gamePath)) {
			a.baseGameDataPath = baseData
		} else {
			a.baseGameDataPath = ""
		}
	}

	p := parser.New(gameVersion, gamePath)
	a.gameWriter = writer.New(gamePath)

	projectileFile, err := parser.ParseProjectileFile(gamePath)
	if err != nil {
		return fmt.Errorf("parse projectiles error: %w", err)
	}
	projRepo := repository.NewProjectile(*projectileFile)
	a.projectileRepo = projRepo
	a.projectileService = service.NewProjectileService(projRepo)
	logger.Info("projectiles loaded", "count", len(projectileFile.Projectiles))

	if gameVersion == config.Medieval {
		gameData, err := p.ParseM2TW()
		if err != nil {
			return err
		}
		repo := repository.NewM2TW(*gameData)
		a.m2twRepo = repo
		a.m2twUnitService = service.NewM2TWUnitService(repo)
		a.m2twBuildingService = service.NewM2TWBuildingService(repo)
		a.rtwRepo = nil
		a.unitService = nil
		a.factionService = nil
		a.buildingService = nil
	} else {
		gameData, err := p.ParseTextFiles()
		if err != nil {
			return err
		}
		repo := repository.New(*gameData)
		a.rtwRepo = repo
		a.unitService = service.NewUnitService(repo)
		a.factionService = service.NewFactionService(repo)
		a.buildingService = service.NewBuildingService(repo)
		a.m2twRepo = nil
		a.m2twUnitService = nil
		a.m2twBuildingService = nil
	}
	logger.Info("game loaded", "version", gameVersion, "path", gamePath)
	return nil
}

func (a *App) GetFactions() []domain.Faction {
	return a.factionService.GetFactions()
}

func (a *App) GetAllUnits() []domain.Unit {
	return a.unitService.GetAllUnits()
}

func (a *App) GetDeletedUnits() []domain.Unit {
	return a.unitService.GetDeletedUnits()
}

func (a *App) GetBuildings() []domain.BuildingGroup {
	return a.buildingService.GetBuildings()
}

func (a *App) UpdateBuildingLevel(groupName, levelName string, slots []domain.RecruitSlot) error {
	logger.OpStart("update_building_level", fmt.Sprintf("%s/%s", groupName, levelName))
	if err := a.buildingService.UpdateBuildingLevel(groupName, levelName, slots); err != nil {
		logger.OpDone("update_building_level", err)
		return err
	}
	err := a.gameWriter.SaveBuildingsDraft(a.rtwRepo.GetData())
	logger.OpDone("update_building_level", err)
	return err
}

func (a *App) UpdateBuildingLevelProps(groupName, levelName string, cost, construction int, settlementMin string, requiredCultures []string, dependencyGroup, dependencyLevel string, upgrades, bonusLines []string) error {
	logger.OpStart("update_building_level_props", fmt.Sprintf("%s/%s", groupName, levelName))
	if err := a.buildingService.UpdateBuildingLevelProps(groupName, levelName, cost, construction, settlementMin, requiredCultures, dependencyGroup, dependencyLevel, upgrades, bonusLines); err != nil {
		logger.OpDone("update_building_level_props", err)
		return err
	}
	err := a.gameWriter.SaveBuildingsDraft(a.rtwRepo.GetData())
	logger.OpDone("update_building_level_props", err)
	return err
}

func (a *App) RevertBuildings() error {
	return a.buildingService.RevertBuildings()
}

func (a *App) GetUnitsByFaction(faction string) (map[string]map[string][]domain.Unit, error) {
	return a.unitService.GetFactionUnitsByCategoryAndClass(faction)
}

func (a *App) GetUnitByType(unitType string) (*domain.Unit, error) {
	return a.unitService.GetUnitByType(unitType)
}

func (a *App) UpdateUnit(originalType string, unit domain.Unit) error {
	logger.OpStart("update_unit", unit.Type)
	if err := a.unitService.Update(originalType, unit); err != nil {
		logger.OpDone("update_unit", err)
		return err
	}
	if originalType != unit.Type && len(unit.Ownership) > 0 {
		_ = a.gameWriter.RenameUnitIcon(originalType, unit.Type, unit.Ownership[0])
	}
	err := a.gameWriter.SaveDraft(a.rtwRepo.GetData(), a.rtwRepo.GetChanges())
	logger.OpDone("update_unit", err)
	return err
}

func (a *App) HasUnsavedChanges() bool {
	return a.unitService.HasUnsavedChanges() || a.projectileService.HasUnsavedChanges()
}

func (a *App) Save() error {
	logger.OpStart("save", "apply to game")
	if err := a.gameWriter.Apply(); err != nil {
		logger.OpDone("save", err)
		return err
	}
	a.rtwRepo.CommitSave()
	a.projectileRepo.CommitSave()
	logger.OpDone("save", nil)
	return nil
}

// Validate возвращает список проблем юнита или пустой срез если всё ок.
func (a *App) Validate(originalType string, unit domain.Unit) []string {
	return a.unitService.Validate(unit, originalType)
}

func (a *App) GetCultureNames() []string {
	return a.buildingService.GetCultureNames()
}

func (a *App) GetProjectileTypes() []string {
	return a.buildingService.GetProjectileTypes()
}

func (a *App) GetUnitBuildings(unitType string) []domain.RecruitLocation {
	return a.buildingService.GetUnitBuilding(unitType)
}

// ── Projectiles (shared between RTW and M2TW) ────────────────────────────────

func (a *App) GetProjectiles() []domain.Projectile {
	return a.projectileService.GetAll()
}

func (a *App) GetProjectileDelays() []domain.ProjectileDelay {
	return a.projectileService.GetDelays()
}

func (a *App) GetProjectileByName(name string) (*domain.Projectile, error) {
	return a.projectileService.GetByName(name)
}

func (a *App) GetProjectileChangeType(name string) string {
	return a.projectileService.GetChangeType(name)
}

func (a *App) UpdateProjectile(originalName string, p domain.Projectile) error {
	logger.OpStart("update_projectile", p.Name)
	if err := a.projectileService.Update(originalName, p); err != nil {
		logger.OpDone("update_projectile", err)
		return err
	}
	err := a.gameWriter.SaveProjectilesDraft(a.projectileRepo.GetData(), a.projectileRepo.GetChanges())
	logger.OpDone("update_projectile", err)
	return err
}

func (a *App) CreateProjectile(templateName, newName string) error {
	logger.OpStart("create_projectile", fmt.Sprintf("%s from %s", newName, templateName))
	if err := a.projectileService.Create(templateName, newName); err != nil {
		logger.OpDone("create_projectile", err)
		return err
	}
	err := a.gameWriter.SaveProjectilesDraft(a.projectileRepo.GetData(), a.projectileRepo.GetChanges())
	logger.OpDone("create_projectile", err)
	return err
}

func (a *App) DeleteProjectile(name string) error {
	logger.OpStart("delete_projectile", name)
	if err := a.projectileService.Delete(name); err != nil {
		logger.OpDone("delete_projectile", err)
		return err
	}
	err := a.gameWriter.SaveProjectilesDraft(a.projectileRepo.GetData(), a.projectileRepo.GetChanges())
	logger.OpDone("delete_projectile", err)
	return err
}

func (a *App) RevertProjectile(name string) error {
	return a.projectileService.Revert(name)
}

func (a *App) RevertAllProjectiles() {
	a.projectileService.RevertAll()
}

func (a *App) CopyUnit(unitType, faction string) (string, error) {
	logger.OpStart("copy_unit", fmt.Sprintf("%s → %s", unitType, faction))
	srcFaction := a.unitService.FirstFaction(unitType)
	soldierModel := a.unitService.SoldierModel(unitType)
	newType, err := a.unitService.CopyUnit(unitType, faction)
	if err != nil {
		logger.OpDone("copy_unit", err)
		return "", err
	}
	_ = a.gameWriter.CopyUnitModelAssets(soldierModel, unitType, newType, srcFaction, faction)
	err = a.gameWriter.SaveDraft(a.rtwRepo.GetData(), a.rtwRepo.GetChanges())
	logger.OpDone("copy_unit", err)
	return newType, err
}

func (a *App) CreateUnit(templateType, newType, faction string) error {
	logger.OpStart("create_unit", fmt.Sprintf("%s from %s", newType, templateType))
	srcFaction := a.unitService.FirstFaction(templateType)
	soldierModel := a.unitService.SoldierModel(templateType)
	if err := a.unitService.CreateUnit(templateType, newType, faction); err != nil {
		logger.OpDone("create_unit", err)
		return err
	}
	_ = a.gameWriter.CopyUnitModelAssets(soldierModel, templateType, newType, srcFaction, faction)
	err := a.gameWriter.SaveDraft(a.rtwRepo.GetData(), a.rtwRepo.GetChanges())
	logger.OpDone("create_unit", err)
	return err
}

func (a *App) GetUnitChangeType(unitType string) string {
	return a.unitService.GetUnitChangeType(unitType)
}

func (a *App) DeleteUnit(unitType, faction string) error {
	logger.OpStart("delete_unit", fmt.Sprintf("%s from %s", unitType, faction))
	if err := a.unitService.Delete(unitType, faction); err != nil {
		logger.OpDone("delete_unit", err)
		return err
	}
	if err := a.gameWriter.SaveDraft(a.rtwRepo.GetData(), a.rtwRepo.GetChanges()); err != nil {
		logger.OpDone("delete_unit", err)
		return err
	}
	var err error
	if a.rtwRepo.IsBuildingsDirty() {
		err = a.gameWriter.SaveBuildingsDraft(a.rtwRepo.GetData())
	}
	logger.OpDone("delete_unit", err)
	return err
}

func (a *App) HardDeleteUnit(unitType string) error {
	logger.OpStart("hard_delete_unit", unitType)
	_ = a.gameWriter.DeleteUnitAssets(unitType)
	if err := a.unitService.HardDelete(unitType); err != nil {
		logger.OpDone("hard_delete_unit", err)
		return err
	}
	if err := a.gameWriter.SaveDraft(a.rtwRepo.GetData(), a.rtwRepo.GetChanges()); err != nil {
		logger.OpDone("hard_delete_unit", err)
		return err
	}
	var err error
	if a.rtwRepo.IsBuildingsDirty() {
		err = a.gameWriter.SaveBuildingsDraft(a.rtwRepo.GetData())
	}
	logger.OpDone("hard_delete_unit", err)
	return err
}

func (a *App) IsCopyUnit(unitType string) bool {
	for _, word := range strings.Fields(strings.ToLower(unitType)) {
		if word == "copy" {
			return true
		}
	}
	return false
}

func (a *App) RevertUnit(unitType string) error {
	return a.unitService.Revert(unitType)
}

func (a *App) RevertAll() {
	a.unitService.RevertAll()
}

// findAsset ищет файл сначала в папке мода, потом в базовой игре.
// Возвращает абсолютный путь и источник ("mod", "base", "").
func (a *App) findAsset(relPath string) (absPath, source string) {
	check := func(root, src string) (string, string) {
		p := filepath.Join(root, relPath)
		if _, err := os.Stat(p); err == nil {
			return p, src
		}
		return "", ""
	}
	if p, s := check(a.gamePath, "mod"); p != "" {
		return p, s
	}
	if a.baseGameDataPath != "" {
		if p, s := check(a.baseGameDataPath, "base"); p != "" {
			return p, s
		}
	}
	return "", ""
}

// iconRelPaths возвращает список relative-путей для иконки юнита (с fallback на короткое имя фракции).
func iconRelPaths(model, faction string) []string {
	filename := "#" + strings.ToLower(model) + ".tga"
	rels := []string{
		filepath.Join("UI", "units", faction, filename),
		filepath.Join("UI", "units", faction, strings.ToUpper(filename)),
	}
	if idx := strings.LastIndex(faction, "_"); idx != -1 {
		short := faction[idx+1:]
		rels = append(rels,
			filepath.Join("UI", "units", short, filename),
			filepath.Join("UI", "units", short, strings.ToUpper(filename)),
		)
	}
	return rels
}

func tgaToBase64PNG(absPath string) string {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}
	img, err := tgadecoder.Decode(data)
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// GetUnitIcon возвращает data:image/png;base64,... или "".
func (a *App) GetUnitIcon(unitType, faction string) string {
	info := a.GetUnitIconInfo(unitType, faction)
	return info.Data
}

// GetUnitIconInfo возвращает иконку + источник (mod/base/"") + relative path.
func (a *App) GetUnitIconInfo(unitType, faction string) domain.IconInfo {
	unit, err := a.unitService.GetUnitByType(unitType)
	if err != nil || unit.Soldier.Model == "" {
		return domain.IconInfo{}
	}
	for _, rel := range iconRelPaths(unit.Soldier.Model, faction) {
		abs, src := a.findAsset(rel)
		if abs == "" {
			continue
		}
		data := tgaToBase64PNG(abs)
		if data == "" {
			continue
		}
		return domain.IconInfo{Data: data, Source: src, RelPath: filepath.ToSlash(rel)}
	}
	// Иконка не найдена — возвращаем только ожидаемый путь
	rels := iconRelPaths(unit.Soldier.Model, faction)
	return domain.IconInfo{RelPath: filepath.ToSlash(rels[0])}
}

// GetDataPaths возвращает пути к данным: mod (активный) и base (базовая игра, если мод выбран).
func (a *App) GetDataPaths() map[string]string {
	result := map[string]string{"mod": a.gamePath}
	if a.baseGameDataPath != "" {
		result["base"] = a.baseGameDataPath
	}
	return result
}

// PickFile открывает нативный диалог выбора файла. Возвращает путь или "".
func (a *App) PickFile(title, filterName, filterPattern string) string {
	path, _ := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: title,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: filterName, Pattern: filterPattern},
		},
	})
	return path
}

// SaveIconFile копирует файл иконки в указанную корневую папку data с правильным именем.
func (a *App) SaveIconFile(unitType, faction, srcPath, destDataRoot string) error {
	unit, err := a.unitService.GetUnitByType(unitType)
	if err != nil {
		return err
	}
	filename := "#" + strings.ToLower(unit.Soldier.Model) + ".tga"
	rel := filepath.Join("UI", "units", faction, filename)
	dst := filepath.Join(destDataRoot, rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	return copyFileRaw(srcPath, dst)
}

// GetUnitModelFiles возвращает CAS-файлы модели юнита из обоих путей.
func (a *App) GetUnitModelFiles(unitType string) []domain.AssetFile {
	unit, err := a.unitService.GetUnitByType(unitType)
	if err != nil || unit.Soldier.Model == "" {
		return nil
	}
	return a.findAssetFiles("models_unit", unit.Soldier.Model)
}

// GetUnitTextureFiles возвращает файлы текстур модели юнита из обоих путей.
func (a *App) GetUnitTextureFiles(unitType string) []domain.AssetFile {
	unit, err := a.unitService.GetUnitByType(unitType)
	if err != nil || unit.Soldier.Model == "" {
		return nil
	}
	return a.findAssetFiles(filepath.Join("models_unit", "textures"), unit.Soldier.Model)
}

// findAssetFiles ищет файлы в subdir, имя которых начинается с modelName (без учёта регистра).
func (a *App) findAssetFiles(subdir, modelName string) []domain.AssetFile {
	seen := map[string]bool{}
	var result []domain.AssetFile
	prefix := strings.ToLower(modelName)

	for _, root := range []struct{ dir, src string }{{a.gamePath, "mod"}, {a.baseGameDataPath, "base"}} {
		if root.dir == "" {
			continue
		}
		dir := filepath.Join(root.dir, subdir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if !strings.HasPrefix(strings.ToLower(e.Name()), prefix) {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(subdir, e.Name()))
			if seen[strings.ToLower(rel)] {
				continue
			}
			seen[strings.ToLower(rel)] = true
			result = append(result, domain.AssetFile{
				Name:    e.Name(),
				RelPath: rel,
				Source:  root.src,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// CopyAssetToRoot копирует файл по абсолютному srcPath в destDataRoot сохраняя relPath.
func (a *App) CopyAssetToRoot(srcAbsPath, destDataRoot, relPath string) error {
	dst := filepath.Join(destDataRoot, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	return copyFileRaw(srcAbsPath, dst)
}

// UploadAssetFile копирует произвольный файл в subdir внутри destDataRoot.
func (a *App) UploadAssetFile(srcPath, destDataRoot, subdir string) error {
	filename := filepath.Base(srcPath)
	dst := filepath.Join(destDataRoot, filepath.FromSlash(subdir), filename)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	return copyFileRaw(srcPath, dst)
}

// AbsAssetPath возвращает абсолютный путь к файлу по его relPath и источнику.
func (a *App) AbsAssetPath(relPath, source string) string {
	root := a.gamePath
	if source == "base" && a.baseGameDataPath != "" {
		root = a.baseGameDataPath
	}
	return filepath.Join(root, filepath.FromSlash(relPath))
}

// ── M2TW API ──────────────────────────────────────────────────────────────────

func (a *App) GetM2TWFactions() []domain.M2TWFaction {
	if a.m2twBuildingService == nil {
		return nil
	}
	return a.m2twBuildingService.GetFactions()
}

func (a *App) GetM2TWAllUnits() []domain.M2TWUnit {
	if a.m2twUnitService == nil {
		return nil
	}
	return a.m2twUnitService.GetAllUnits()
}

func (a *App) GetM2TWDeletedUnits() []domain.M2TWUnit {
	if a.m2twUnitService == nil {
		return nil
	}
	return a.m2twUnitService.GetDeletedUnits()
}

func (a *App) GetM2TWUnitsByFaction(faction string) (map[string]map[string][]domain.M2TWUnit, error) {
	if a.m2twUnitService == nil {
		return nil, fmt.Errorf("M2TW not loaded")
	}
	return a.m2twUnitService.GetFactionUnitsByCategoryAndClass(faction)
}

func (a *App) GetM2TWUnitByType(unitType string) (*domain.M2TWUnit, error) {
	if a.m2twUnitService == nil {
		return nil, fmt.Errorf("M2TW not loaded")
	}
	return a.m2twUnitService.GetUnitByType(unitType)
}

func (a *App) UpdateM2TWUnit(originalType string, unit domain.M2TWUnit) error {
	if a.m2twUnitService == nil {
		return fmt.Errorf("M2TW not loaded")
	}
	logger.OpStart("update_m2tw_unit", unit.Type)
	if err := a.m2twUnitService.Update(originalType, unit); err != nil {
		logger.OpDone("update_m2tw_unit", err)
		return err
	}
	if originalType != unit.Type && len(unit.Ownership) > 0 {
		_ = a.gameWriter.RenameUnitCardDraft(originalType, unit.Type, unit.Ownership[0])
	}
	err := a.gameWriter.SaveM2TWDraft(a.m2twRepo.GetData(), a.m2twRepo.GetChanges())
	logger.OpDone("update_m2tw_unit", err)
	return err
}

func (a *App) CopyM2TWUnit(unitType, faction string) (string, error) {
	if a.m2twUnitService == nil {
		return "", fmt.Errorf("M2TW not loaded")
	}
	logger.OpStart("copy_m2tw_unit", fmt.Sprintf("%s → %s", unitType, faction))
	newType, srcFaction, soldierModel, err := a.m2twUnitService.CopyUnit(unitType, faction)
	if err != nil {
		logger.OpDone("copy_m2tw_unit", err)
		return "", err
	}
	_ = a.gameWriter.CopyBattleModelDraft(unitType, newType)
	_ = a.gameWriter.PatchSoldierFactionDraft(soldierModel, srcFaction, faction)
	_ = a.gameWriter.CopyUnitCardDraft(unitType, newType, srcFaction, faction)
	err = a.gameWriter.SaveM2TWDraft(a.m2twRepo.GetData(), a.m2twRepo.GetChanges())
	logger.OpDone("copy_m2tw_unit", err)
	return newType, err
}

func (a *App) CreateM2TWUnit(templateType, newType, faction string) error {
	if a.m2twUnitService == nil {
		return fmt.Errorf("M2TW not loaded")
	}
	logger.OpStart("create_m2tw_unit", fmt.Sprintf("%s from %s", newType, templateType))
	srcFaction, soldierModel, err := a.m2twUnitService.CreateUnit(templateType, newType, faction)
	if err != nil {
		logger.OpDone("create_m2tw_unit", err)
		return err
	}
	_ = a.gameWriter.CopyBattleModelDraft(templateType, newType)
	_ = a.gameWriter.PatchSoldierFactionDraft(soldierModel, srcFaction, faction)
	_ = a.gameWriter.CopyUnitCardDraft(templateType, newType, srcFaction, faction)
	err = a.gameWriter.SaveM2TWDraft(a.m2twRepo.GetData(), a.m2twRepo.GetChanges())
	logger.OpDone("create_m2tw_unit", err)
	return err
}

func (a *App) DeleteM2TWUnit(unitType, faction string) error {
	if a.m2twUnitService == nil {
		return fmt.Errorf("M2TW not loaded")
	}
	logger.OpStart("delete_m2tw_unit", fmt.Sprintf("%s from %s", unitType, faction))
	if err := a.m2twUnitService.Delete(unitType, faction); err != nil {
		logger.OpDone("delete_m2tw_unit", err)
		return err
	}
	if err := a.gameWriter.SaveM2TWDraft(a.m2twRepo.GetData(), a.m2twRepo.GetChanges()); err != nil {
		logger.OpDone("delete_m2tw_unit", err)
		return err
	}
	var err error
	if a.m2twRepo.IsBuildingsDirty() {
		err = a.gameWriter.SaveM2TWBuildingsDraft(a.m2twRepo.GetData())
	}
	logger.OpDone("delete_m2tw_unit", err)
	return err
}

func (a *App) HardDeleteM2TWUnit(unitType string) error {
	if a.m2twUnitService == nil {
		return fmt.Errorf("M2TW not loaded")
	}
	logger.OpStart("hard_delete_m2tw_unit", unitType)
	_ = a.gameWriter.DeleteUnitAssets(unitType)
	_ = a.gameWriter.DeleteBattleModelDraft(unitType)
	if err := a.m2twUnitService.HardDelete(unitType); err != nil {
		logger.OpDone("hard_delete_m2tw_unit", err)
		return err
	}
	if err := a.gameWriter.SaveM2TWDraft(a.m2twRepo.GetData(), a.m2twRepo.GetChanges()); err != nil {
		logger.OpDone("hard_delete_m2tw_unit", err)
		return err
	}
	var err error
	if a.m2twRepo.IsBuildingsDirty() {
		err = a.gameWriter.SaveM2TWBuildingsDraft(a.m2twRepo.GetData())
	}
	logger.OpDone("hard_delete_m2tw_unit", err)
	return err
}

func (a *App) RevertM2TWUnit(unitType string) error {
	if a.m2twUnitService == nil {
		return fmt.Errorf("M2TW not loaded")
	}
	return a.m2twUnitService.Revert(unitType)
}

func (a *App) RevertM2TWAll() {
	if a.m2twUnitService != nil {
		a.m2twUnitService.RevertAll()
	}
}

func (a *App) GetM2TWUnitChangeType(unitType string) string {
	if a.m2twUnitService == nil {
		return "none"
	}
	return a.m2twUnitService.GetUnitChangeType(unitType)
}

func (a *App) ValidateM2TWUnit(originalType string, unit domain.M2TWUnit) []string {
	if a.m2twUnitService == nil {
		return nil
	}
	return a.m2twUnitService.Validate(unit, originalType)
}

func (a *App) M2TWHasUnsavedChanges() bool {
	return (a.m2twUnitService != nil && a.m2twUnitService.HasUnsavedChanges()) ||
		(a.projectileService != nil && a.projectileService.HasUnsavedChanges())
}

func (a *App) SaveM2TW() error {
	if a.m2twUnitService == nil {
		return fmt.Errorf("M2TW not loaded")
	}
	logger.OpStart("save_m2tw", "apply to game")
	if err := a.gameWriter.Apply(); err != nil {
		logger.OpDone("save_m2tw", err)
		return err
	}
	a.m2twRepo.CommitSave()
	a.projectileRepo.CommitSave()
	logger.OpDone("save_m2tw", nil)
	return nil
}

func (a *App) GetM2TWBuildings() []domain.M2TWBuildingGroup {
	if a.m2twBuildingService == nil {
		return nil
	}
	return a.m2twBuildingService.GetBuildings()
}

func (a *App) GetM2TWFactionBuildings(faction string) []domain.M2TWBuildingGroup {
	if a.m2twBuildingService == nil {
		return nil
	}
	return a.m2twBuildingService.GetFactionBuildings(faction)
}

func (a *App) UpdateM2TWBuildingLevel(groupName, levelName string, pools []domain.M2TWRecruitPool, bonusLines []string) error {
	if a.m2twBuildingService == nil {
		return fmt.Errorf("M2TW not loaded")
	}
	logger.OpStart("update_m2tw_building_level", fmt.Sprintf("%s/%s", groupName, levelName))
	if err := a.m2twBuildingService.UpdateBuildingLevel(groupName, levelName, pools, bonusLines); err != nil {
		logger.OpDone("update_m2tw_building_level", err)
		return err
	}
	err := a.gameWriter.SaveM2TWBuildingsDraft(a.m2twRepo.GetData())
	logger.OpDone("update_m2tw_building_level", err)
	return err
}

func (a *App) UpdateM2TWBuildingLevelProps(groupName, levelName string, cost, construction, convertTo int, settlementMin, settlementType string, requiredFactions []string, dependencyGroup, dependencyLevel string, upgrades []string) error {
	if a.m2twBuildingService == nil {
		return fmt.Errorf("M2TW not loaded")
	}
	logger.OpStart("update_m2tw_building_level_props", fmt.Sprintf("%s/%s", groupName, levelName))
	if err := a.m2twBuildingService.UpdateBuildingLevelProps(groupName, levelName, cost, construction, convertTo, settlementMin, settlementType, requiredFactions, dependencyGroup, dependencyLevel, upgrades); err != nil {
		logger.OpDone("update_m2tw_building_level_props", err)
		return err
	}
	err := a.gameWriter.SaveM2TWBuildingsDraft(a.m2twRepo.GetData())
	logger.OpDone("update_m2tw_building_level_props", err)
	return err
}

func (a *App) RevertM2TWBuildings() error {
	if a.m2twBuildingService == nil {
		return fmt.Errorf("M2TW not loaded")
	}
	return a.m2twBuildingService.RevertBuildings()
}

func (a *App) GetM2TWReligions() []string {
	if a.m2twBuildingService == nil {
		return nil
	}
	return a.m2twBuildingService.GetReligions()
}

func (a *App) GetM2TWHiddenResources() []string {
	if a.m2twBuildingService == nil {
		return nil
	}
	return a.m2twBuildingService.GetHiddenResources()
}

func (a *App) GetM2TWProjectileTypes() []string {
	if a.m2twBuildingService == nil {
		return nil
	}
	return a.m2twBuildingService.GetProjectileTypes()
}

func (a *App) GetM2TWUnitBuildings(unitType string) []domain.RecruitLocation {
	if a.m2twBuildingService == nil {
		return nil
	}
	return a.m2twBuildingService.GetUnitBuildings(unitType)
}

func copyFileRaw(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
