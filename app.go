package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"modding-utils/internal/config"
	"modding-utils/internal/domain"
	"modding-utils/internal/parser"
	"modding-utils/internal/repository"
	"modding-utils/internal/service"
	"modding-utils/internal/tgadecoder"
	"modding-utils/internal/writer"
	"os"
	"path/filepath"
	"strings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type AppConfig struct {
	Game            int    `json:"game"`
	GamePath        string `json:"gamePath"`
	SelectedModPath string `json:"selectedModPath"`
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
	path, err := appConfigPath()
	if err != nil {
		return err
	}
	cfg := AppConfig{Game: game, GamePath: gamePath, SelectedModPath: selectedModPath}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

type App struct {
	ctx             context.Context
	gamePath        string
	generalService  *service.GeneralService
	unitService     *service.UnitService
	factionService  *service.FactionService
	buildingService *service.BuildingService
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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

// AddModPath добавляет папку мода вручную (для RTW).
func (a *App) AddModPath(path, name string) error {
	return a.generalService.AddModPath(path, name)
}

// InitGame вызывается с фронтенда когда пользователь выбрал путь к игре.
func (a *App) InitGame(gamePath string, gameVersion config.GameVersion) error {
	p := parser.New(gameVersion, gamePath)
	gameData, err := p.ParseTextFiles()
	if err != nil {
		return err
	}

	w := writer.New(gamePath)
	repo := repository.New(*gameData, w)
	a.gamePath = gamePath
	a.unitService = service.NewUnitService(repo)
	a.factionService = service.NewFactionService(repo)
	a.buildingService = service.NewBuildingService(repo)
	return nil
}

func (a *App) GetFactions() []domain.Faction {
	return a.factionService.GetFactions()
}

func (a *App) GetAllUnits() []domain.Unit {
	return a.unitService.GetAllUnits()
}

func (a *App) GetBuildings() []domain.BuildingGroup {
	return a.buildingService.GetBuildings()
}

func (a *App) UpdateBuildingLevel(groupName, levelName string, slots []domain.RecruitSlot) error {
	return a.buildingService.UpdateBuildingLevel(groupName, levelName, slots)
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
	return a.unitService.Update(originalType, unit)
}

func (a *App) HasUnsavedChanges() bool {
	return a.unitService.HasUnsavedChanges()
}

func (a *App) Save() error {
	return a.unitService.Save()
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

func (a *App) CopyUnit(unitType, faction string) (string, error) {
	return a.unitService.CopyUnit(unitType, faction)
}

func (a *App) GetUnitChangeType(unitType string) string {
	return a.unitService.GetUnitChangeType(unitType)
}

func (a *App) RevertUnit(unitType string) error {
	return a.unitService.Revert(unitType)
}

func (a *App) RevertAll() {
	a.unitService.RevertAll()
}

// GetUnitIcon возвращает data:image/png;base64,... или "" если иконка не найдена.
// RTW хранит иконки в UI/units/{faction}/#{model}.tga
func (a *App) GetUnitIcon(unitType, faction string) string {
	unit, err := a.unitService.GetUnitByType(unitType)
	if err != nil || unit.Soldier.Model == "" {
		return ""
	}
	filename := "#" + strings.ToLower(unit.Soldier.Model) + ".tga"

	candidates := []string{
		filepath.Join(a.gamePath, "UI", "units", faction, filename),
		filepath.Join(a.gamePath, "UI", "units", faction, strings.ToUpper(filename)),
	}
	// для romans_julii → julii
	if idx := strings.LastIndex(faction, "_"); idx != -1 {
		short := faction[idx+1:]
		candidates = append(candidates,
			filepath.Join(a.gamePath, "UI", "units", short, filename),
			filepath.Join(a.gamePath, "UI", "units", short, strings.ToUpper(filename)),
		)
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		img, err := tgadecoder.Decode(data)
		if err != nil {
			continue
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			continue
		}
		return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	}
	return ""
}
