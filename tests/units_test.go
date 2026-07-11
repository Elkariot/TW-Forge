package tests

import (
	"os"
	"strings"
	"testing"

	"tw-forge/internal/config"
	"tw-forge/internal/parser"
	"tw-forge/internal/repository"
	"tw-forge/internal/service"
	"tw-forge/internal/writer"
)

// ── RTW: AddFactionToUnit + asset patching ──────────────────────────────────

func TestAddFactionToUnit_RTW_Unowned(t *testing.T) {
	dataPath := setupModDir(t, rtwFixture)
	data, err := parser.New(config.Rome, dataPath).ParseTextFiles()
	if err != nil {
		t.Fatalf("ParseTextFiles: %v", err)
	}
	repo := repository.New(*data)
	unitSvc := service.NewUnitService(repo)

	unit, ok := repo.GetUnitByType("merc free company")
	if !ok || len(unit.Ownership) != 0 {
		t.Fatalf("fixture assumption broken: merc free company ownership = %v, want empty", unit.Ownership)
	}

	srcFaction, soldierModel, err := unitSvc.AddFactionToUnit("merc free company", "carthage")
	if err != nil {
		t.Fatalf("AddFactionToUnit: %v", err)
	}
	if srcFaction != "" {
		t.Errorf("srcFaction = %q, want empty (unit had no prior owner)", srcFaction)
	}
	if soldierModel != "free_company_merc" {
		t.Errorf("soldierModel = %q, want free_company_merc", soldierModel)
	}

	updated, _ := repo.GetUnitByType("merc free company")
	if len(updated.Ownership) != 1 || updated.Ownership[0] != "carthage" {
		t.Errorf("Ownership after add = %v, want [carthage]", updated.Ownership)
	}

	gw := writer.New(dataPath)
	if err := gw.CopyUnitModelAssets(soldierModel, "merc free company", "merc free company", srcFaction, "carthage"); err != nil {
		t.Fatalf("CopyUnitModelAssets: %v", err)
	}
	// No draft descr_model_battle.txt should be produced: srcFaction is empty, so
	// CopyUnitModelAssets no-ops before touching any file (see its early guard).
	if _, err := os.Stat(dataPath + "/../_modding_editor/draft/descr_model_battle.txt"); err == nil {
		t.Errorf("descr_model_battle.txt draft was created even though srcFaction was empty")
	}
}

func TestAddFactionToUnit_RTW_Owned(t *testing.T) {
	dataPath := setupModDir(t, rtwFixture)
	data, err := parser.New(config.Rome, dataPath).ParseTextFiles()
	if err != nil {
		t.Fatalf("ParseTextFiles: %v", err)
	}
	repo := repository.New(*data)
	unitSvc := service.NewUnitService(repo)

	srcFaction, soldierModel, err := unitSvc.AddFactionToUnit("carthaginian peltast", "gauls")
	if err != nil {
		t.Fatalf("AddFactionToUnit: %v", err)
	}
	if srcFaction != "carthage" {
		t.Fatalf("srcFaction = %q, want carthage", srcFaction)
	}

	gw := writer.New(dataPath)
	if err := gw.CopyUnitModelAssets(soldierModel, "carthaginian peltast", "carthaginian peltast", srcFaction, "gauls"); err != nil {
		t.Fatalf("CopyUnitModelAssets: %v", err)
	}

	draft, err := os.ReadFile(dataPath + "/../_modding_editor/draft/descr_model_battle.txt")
	if err != nil {
		t.Fatalf("read patched descr_model_battle.txt: %v", err)
	}
	content := string(draft)
	block := extractTypeBlock(content, "carthaginian_peltast")
	if !strings.Contains(block, "texture\t\t\tgauls,") && !strings.Contains(block, "texture gauls,") && !strings.Contains(block, "texture\tgauls,") {
		if !strings.Contains(block, "gauls,") {
			t.Errorf("carthaginian_peltast block missing a cloned 'gauls' texture line:\n%s", block)
		}
	}
}

func extractTypeBlock(content, typeName string) string {
	idx := strings.Index(content, "type\t\t\t\t"+typeName)
	if idx == -1 {
		idx = strings.Index(content, "type "+typeName)
	}
	if idx == -1 {
		return ""
	}
	rest := content[idx:]
	if next := strings.Index(rest[1:], "\ntype"); next != -1 {
		return rest[:next+1]
	}
	return rest
}

// ── M2TW: AddFactionToUnit + modeldb patching ───────────────────────────────

func TestAddFactionToUnit_M2TW_Owned(t *testing.T) {
	dataPath := setupModDir(t, m2twFixture)
	data, err := parser.New(config.Medieval, dataPath).ParseM2TW()
	if err != nil {
		t.Fatalf("ParseM2TW: %v", err)
	}
	repo := repository.NewM2TW(*data)
	unitSvc := service.NewM2TWUnitService(repo)

	srcFaction, soldierModel, err := unitSvc.AddFactionToUnit("Merc English Longbowmen", "byzantium")
	if err != nil {
		t.Fatalf("AddFactionToUnit: %v", err)
	}
	if srcFaction != "england" {
		t.Fatalf("srcFaction = %q, want england", srcFaction)
	}
	if soldierModel != "english_archers_merc" {
		t.Fatalf("soldierModel = %q, want english_archers_merc", soldierModel)
	}

	updated, _ := repo.GetUnitByType("Merc English Longbowmen")
	if !containsStr(updated.Ownership, "byzantium") {
		t.Errorf("Ownership after add = %v, want to contain byzantium", updated.Ownership)
	}
	if !containsStr(updated.Eras["0"], "byzantium") {
		t.Errorf("Eras[0] after add = %v, want to contain byzantium", updated.Eras["0"])
	}

	gw := writer.New(dataPath)
	if err := gw.PatchSoldierFactionDraft(soldierModel, srcFaction, "byzantium"); err != nil {
		t.Fatalf("PatchSoldierFactionDraft: %v", err)
	}

	db, err := parser.ParseBattleModels(dataPath + "/../_modding_editor/draft/unit_models/battle_models.modeldb")
	if err != nil {
		t.Fatalf("re-parse patched modeldb: %v", err)
	}
	idx, ok := db.Index[soldierModel]
	if !ok {
		t.Fatalf("soldier model %q missing from patched modeldb", soldierModel)
	}
	if !strings.Contains(db.Models[idx].RawBlock, "byzantium") {
		t.Errorf("patched modeldb entry doesn't mention byzantium:\n%s", db.Models[idx].RawBlock)
	}
}

// TestAddFactionToUnit_M2TW_CaseMismatch documents a real-data quirk found while
// building these fixtures: vanilla/VK EDU capitalizes soldier model names (e.g.
// "Cuman_Horse_Archers") but battle_models.modeldb stores them lowercase
// ("cuman_horse_archers"). Since db.Index is keyed by the exact string found in
// the file, a soldierModel taken verbatim from EDU never matches, and
// PatchSoldierFactionDraft silently no-ops for every vanilla-authored unit.
func TestAddFactionToUnit_M2TW_CaseMismatch(t *testing.T) {
	dataPath := setupModDir(t, m2twFixture)

	db, err := parser.ParseBattleModels(dataPath + "/unit_models/battle_models.modeldb")
	if err != nil {
		t.Fatalf("ParseBattleModels: %v", err)
	}
	lower := "english_archers_merc"
	upper := "English_Archers_Merc"
	if _, ok := db.Index[lower]; !ok {
		t.Fatalf("fixture assumption broken: %q not indexed", lower)
	}
	if _, ok := db.Index[upper]; ok {
		t.Fatalf("fixture assumption broken: %q unexpectedly indexed (should only be lowercase)", upper)
	}

	gw := writer.New(dataPath)
	// Simulate what happens with vanilla-cased EDU data: soldierModel passed in
	// with the capitalized form that real files actually use.
	if err := gw.PatchSoldierFactionDraft(upper, "england", "byzantium"); err != nil {
		t.Fatalf("PatchSoldierFactionDraft: %v", err)
	}
	if _, err := os.Stat(dataPath + "/../_modding_editor/draft/unit_models/battle_models.modeldb"); err == nil {
		t.Errorf("a draft modeldb was written even though the capitalized key can't match — " +
			"confirms PatchSoldierFactionDraft silently no-ops on real-world casing")
	}
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
