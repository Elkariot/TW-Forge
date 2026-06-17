package main

import (
	"context"
	"modding-utils/internal/config"
	"modding-utils/internal/domain"
	"modding-utils/internal/parser"
	"modding-utils/internal/repository"
	"modding-utils/internal/service"
	"modding-utils/internal/writer"
)

type App struct {
	ctx         context.Context
	unitService *service.UnitService
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
	a.unitService = service.NewUnitService(repo)
	return nil
}

func (a *App) GetFactions() []domain.Faction {
	return a.unitService.GetFactions()
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

func (a *App) RevertUnit(unitType string) error {
	return a.unitService.Revert(unitType)
}

func (a *App) RevertAll() {
	a.unitService.RevertAll()
}
