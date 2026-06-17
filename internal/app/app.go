package app

import (
	"fmt"
	"modding-utils/internal/config"
	"modding-utils/internal/parser"
)

func Start() {
	fmt.Println("Application start")
	gamePath := "./test-data/data"
	gameVersion := config.Rome

	p := parser.New(gameVersion, gamePath)
	gameData, err := p.ParseTextFiles()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Factions: %d\n", len(gameData.Factions))
	fmt.Printf("Units: %d\n", len(gameData.Units))
	fmt.Printf("Buildings: %d\n", len(gameData.Buildings))
}
