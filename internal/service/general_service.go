package service

import (
	"errors"
	"fmt"
	"tw-forge/internal/config"
	"tw-forge/internal/repository"
	"os"
	"path/filepath"
)

type GeneralService struct {
	repo           repository.GameRepository
	gameRoot       string
	gamePath       string
	versionsPath   string
	game           config.GameVersion
	modsPath       map[string]string
	targetFileName string
}

func NewGeneralService(repo repository.GameRepository) *GeneralService {
	return &GeneralService{
		repo:           repo,
		targetFileName: "export_descr_unit.txt",
		modsPath:       make(map[string]string),
	}
}

type GameSettings struct {
	GamePath string
	Game     config.GameVersion
}

func (s *GeneralService) GetGameSettings() GameSettings {
	return GameSettings{
		GamePath: s.gamePath,
		Game:     s.game,
	}
}

func (s *GeneralService) GetMods() map[string]string {
	return s.modsPath
}

// SetGamePath переключает активный путь к данным на указанный мод.
func (s *GeneralService) SetGamePath(modName string) error {
	dataPath, ok := s.modsPath[modName]
	if !ok {
		return fmt.Errorf("mod not found: %s", modName)
	}
	s.gamePath = dataPath
	return nil
}

var rtwExeNames = []string{"RomeTW.exe", "RomeTW-BI.exe", "RomeTW-Alexander.exe"}

func (s *GeneralService) InitGameFolder(game config.GameVersion, userPath string) error {
	if ok, err := s.checkPath(userPath); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("path not valid")
	}

	exeFound := false
	if game == config.Rome {
		for _, exe := range rtwExeNames {
			if ok, _ := s.checkPath(filepath.Join(userPath, exe)); ok {
				exeFound = true
				break
			}
		}
	} else {
		for _, exe := range []string{"medieval2.exe", "kingdoms.exe"} {
			if ok, _ := s.checkPath(filepath.Join(userPath, exe)); ok {
				exeFound = true
				break
			}
		}
		if !exeFound {
			exeFound, _ = s.checkPath(filepath.Join(userPath, "medieval2.preference.cfg"))
		}
	}
	if !exeFound {
		return fmt.Errorf("can't find game exe")
	}

	dataPath := filepath.Join(userPath, "data")
	if ok, err := s.checkPath(dataPath); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("can't find game data folder")
	}

	if ok, err := s.checkPath(filepath.Join(dataPath, s.targetFileName)); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("can't find game data files")
	}

	s.gameRoot = userPath
	s.gamePath = dataPath
	s.game = game
	s.versionsPath = filepath.Join(dataPath, "versions")
	s.modsPath = make(map[string]string) // сбрасываем моды предыдущей игры

	return nil
}

func (s *GeneralService) CheckModsFolder() error {
	if s.game != config.Medieval {
		return nil
	}

	modsFolder := filepath.Join(s.gameRoot, "mods")
	if ok, err := s.checkPath(modsFolder); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("mods folder not found")
	}

	files, err := os.ReadDir(modsFolder)
	if err != nil {
		return fmt.Errorf("error reading mods directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		modRoot := filepath.Join(modsFolder, file.Name())
		if dataPath, ok := s.findModDataPath(modRoot); ok {
			s.modsPath[file.Name()] = dataPath
		}
	}

	return nil
}

func (s *GeneralService) AddModPath(userPath, modName string) error {
	if ok, err := s.checkPath(userPath); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("path not valid")
	}

	dataPath, ok := s.findModDataPath(userPath)
	if !ok {
		return fmt.Errorf("no valid mod data found at path")
	}

	s.modsPath[modName] = dataPath
	return nil
}

// findModDataPath ищет EDU-файл в root напрямую или в root/data.
// Возвращает путь к папке с данными и true если найдено.
func (s *GeneralService) findModDataPath(root string) (string, bool) {
	if ok, _ := s.checkPath(filepath.Join(root, s.targetFileName)); ok {
		return root, true
	}
	dataPath := filepath.Join(root, "data")
	if ok, _ := s.checkPath(filepath.Join(dataPath, s.targetFileName)); ok {
		return dataPath, true
	}
	return "", false
}

func (s *GeneralService) checkPath(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("error while checking path: %w", err)
	}
	return true, nil
}
