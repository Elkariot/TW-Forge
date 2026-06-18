package main

import (
	"bytes"
	"context"
	"encoding/base64"
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

)

type App struct {
	ctx             context.Context
	gamePath        string
	unitService     *service.UnitService
	factionService  *service.FactionService
	buildingService *service.BuildingService
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// TODO: убрать когда фронтенд будет вызывать InitGame с путём от пользователя
	_ = a.InitGame("./test-data/data", config.Rome)
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
