package tests

import (
	"os"
	"testing"

	"tw-forge/internal/config"
	"tw-forge/internal/parser"
)

// These are broad parse-and-sanity-check smoke tests: they exist to catch fixture
// or parser regressions across the file types the mercenaries fixtures also carry
// (EDU, EDB, projectiles, faction lists), not to exhaustively test those features.

func TestSmoke_RTW(t *testing.T) {
	data, err := parser.New(config.Rome, rtwFixture).ParseTextFiles()
	if err != nil {
		t.Fatalf("ParseTextFiles: %v", err)
	}

	if len(data.Units) != 6 {
		t.Errorf("units = %d, want 6", len(data.Units))
	}
	if len(data.Factions) != 3 {
		t.Errorf("factions = %d, want 3", len(data.Factions))
	}
	if len(data.Buildings) != 1 || len(data.Buildings[0].Levels) != 2 {
		t.Fatalf("buildings = %+v, want 1 group with 2 levels", data.Buildings)
	}
	for _, lvl := range data.Buildings[0].Levels {
		if len(lvl.RecruitSlots) != 3 {
			t.Errorf("level %s has %d recruit slots, want 3", lvl.Name, len(lvl.RecruitSlots))
		}
	}

	proj, err := parser.ParseProjectileFile(rtwFixture)
	if err != nil {
		t.Fatalf("ParseProjectileFile: %v", err)
	}
	if len(proj.Projectiles) != 3 {
		t.Fatalf("projectiles = %d, want 3", len(proj.Projectiles))
	}
	foundFiery := false
	for _, p := range proj.Projectiles {
		if p.Name == "fiery_boulder" {
			foundFiery = true
			if p.FlamingOf != "boulder" {
				t.Errorf("fiery_boulder.FlamingOf = %q, want boulder", p.FlamingOf)
			}
		}
	}
	if !foundFiery {
		t.Errorf("fiery_boulder projectile not found")
	}
}

func TestSmoke_M2TW(t *testing.T) {
	data, err := parser.New(config.Medieval, m2twFixture).ParseM2TW()
	if err != nil {
		t.Fatalf("ParseM2TW: %v", err)
	}

	if len(data.Units) != 6 {
		t.Errorf("units = %d, want 6", len(data.Units))
	}
	if len(data.Factions) != 3 {
		t.Errorf("factions = %d, want 3", len(data.Factions))
	}
	for _, f := range data.Factions {
		if f.Religion == "" {
			t.Errorf("faction %s has no religion", f.Name)
		}
	}

	if len(data.Buildings) != 1 || len(data.Buildings[0].Levels) != 2 {
		t.Fatalf("buildings = %+v, want 1 group with 2 levels", data.Buildings)
	}
	for _, lvl := range data.Buildings[0].Levels {
		if len(lvl.RecruitPools) != 3 {
			t.Errorf("level %s has %d recruit pools, want 3", lvl.Name, len(lvl.RecruitPools))
		}
		if lvl.SettlementType != "castle" {
			t.Errorf("level %s SettlementType = %q, want castle", lvl.Name, lvl.SettlementType)
		}
	}

	proj, err := parser.ParseProjectileFile(m2twFixture)
	if err != nil {
		t.Fatalf("ParseProjectileFile: %v", err)
	}
	if len(proj.Projectiles) != 3 {
		t.Fatalf("projectiles = %d, want 3", len(proj.Projectiles))
	}
	if len(proj.Delays) != 3 {
		t.Fatalf("delays = %d, want 3", len(proj.Delays))
	}
}

// TestSmoke_MissingOptionalFiles reproduces a vanilla install: projectiles
// live packed in a .pak (no loose descr_projectile(_new).txt) and some games/mods
// have no export_descr_buildings.txt at all. Neither should fail the top-level
// parse — the corresponding editor tab is just empty (see app.go's
// ProjectilesAvailable / the buildings-parser not-exist handling).
func TestSmoke_MissingOptionalFiles(t *testing.T) {
	dataPath := setupModDir(t, m2twFixture)
	if err := os.Remove(dataPath + "/export_descr_buildings.txt"); err != nil {
		t.Fatalf("remove EDB: %v", err)
	}
	if err := os.Remove(dataPath + "/descr_projectile.txt"); err != nil {
		t.Fatalf("remove projectiles: %v", err)
	}

	data, err := parser.New(config.Medieval, dataPath).ParseM2TW()
	if err != nil {
		t.Fatalf("ParseM2TW should not fail when EDB is missing: %v", err)
	}
	if len(data.Buildings) != 0 {
		t.Errorf("Buildings = %v, want empty", data.Buildings)
	}

	if _, err := parser.ParseProjectileFile(dataPath); err == nil {
		t.Errorf("ParseProjectileFile should still report an error when no file exists — " +
			"app.go.InitGame relies on this to leave projectileService nil")
	}
}
