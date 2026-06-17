package main

import (
	"context"
	"modding-utils/internal/config"
	"modding-utils/internal/domain"
	"modding-utils/internal/parser"
	"modding-utils/internal/repository"
	"modding-utils/internal/service"
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

	p := parser.New(config.Rome, "./test-data/data")
	gameData, err := p.ParseTextFiles()
	if err != nil {
		// TODO: показать ошибку пользователю через Wails dialog
		return
	}

	repo := repository.New(*gameData, nil)
	a.unitService = service.NewUnitService(repo)
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
