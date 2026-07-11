package tests

import (
	"os"
	"testing"

	"tw-forge/internal/domain"
	"tw-forge/internal/parser"
	"tw-forge/internal/repository"
	"tw-forge/internal/service"
	"tw-forge/internal/writer"
)

func findPool(t *testing.T, pools []domain.MercenaryPool, name string) domain.MercenaryPool {
	t.Helper()
	for _, p := range pools {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("pool %q not found", name)
	return domain.MercenaryPool{}
}

// ── Parser ───────────────────────────────────────────────────────────────────

func TestMercenaries_ParseRTW(t *testing.T) {
	campaigns := parser.ListMercenaryCampaigns(rtwFixture)
	if len(campaigns) != 1 || campaigns[0] != campaignName {
		t.Fatalf("ListMercenaryCampaigns = %v, want [%s]", campaigns, campaignName)
	}

	file, err := parser.ParseMercenaryFile(parser.MercenariesFilePath(rtwFixture, campaignName))
	if err != nil {
		t.Fatalf("ParseMercenaryFile: %v", err)
	}
	if len(file.Pools) != 3 {
		t.Fatalf("got %d pools, want 3", len(file.Pools))
	}

	regions, err := parser.ParseRegionNames(parser.RegionsFilePath(rtwFixture, campaignName))
	if err != nil {
		t.Fatalf("ParseRegionNames: %v", err)
	}
	if len(regions) != 3 {
		t.Fatalf("got %d regions, want 3: %v", len(regions), regions)
	}

	greek := findPool(t, file.Pools, "Greek_Pool")
	if len(greek.Units) != 1 {
		t.Fatalf("Greek_Pool has %d units, want 1", len(greek.Units))
	}
	u := greek.Units[0]
	if u.Armour == nil || *u.Armour != 1 {
		t.Errorf("Greek_Pool unit Armour = %v, want 1 (REX field)", u.Armour)
	}
	if u.WeaponLvl == nil || *u.WeaponLvl != 1 {
		t.Errorf("Greek_Pool unit WeaponLvl = %v, want 1 (REX field)", u.WeaponLvl)
	}
}

func TestMercenaries_ParseM2TW(t *testing.T) {
	file, err := parser.ParseMercenaryFile(parser.MercenariesFilePath(m2twFixture, campaignName))
	if err != nil {
		t.Fatalf("ParseMercenaryFile: %v", err)
	}
	if len(file.Pools) != 3 {
		t.Fatalf("got %d pools, want 3", len(file.Pools))
	}

	england := findPool(t, file.Pools, "England_Pool")
	u := england.Units[0]
	if u.Name != "Merc English Longbowmen," {
		t.Errorf("unit name = %q, want the raw comma-suffixed form", u.Name)
	}
	if !u.Crusading {
		t.Errorf("Crusading = false, want true")
	}
	if len(u.Religions) != 1 || u.Religions[0] != "catholic" {
		t.Errorf("Religions = %v, want [catholic]", u.Religions)
	}

	byz := findPool(t, file.Pools, "Byzantium_Pool")
	bu := byz.Units[0]
	if bu.Armour == nil || *bu.Armour != 1 || bu.WeaponLvl == nil || *bu.WeaponLvl != 1 {
		t.Errorf("Byzantium_Pool unit REX/M2EX fields not parsed: armour=%v weaponlvl=%v", bu.Armour, bu.WeaponLvl)
	}
	if bu.StartYear == nil || *bu.StartYear != 1150 || bu.EndYear == nil || *bu.EndYear != 1300 {
		t.Errorf("Byzantium_Pool unit start/end year not parsed: start=%v end=%v", bu.StartYear, bu.EndYear)
	}

	lith := findPool(t, file.Pools, "Lithuania_Pool")
	lu := lith.Units[0]
	if len(lu.Events) != 1 || lu.Events[0] != "matchlock" {
		t.Errorf("Lithuania_Pool unit Events = %v, want [matchlock]", lu.Events)
	}
}

// ── Repository + Service CRUD ───────────────────────────────────────────────

func TestMercenaries_ServiceCRUD(t *testing.T) {
	file, err := parser.ParseMercenaryFile(parser.MercenariesFilePath(rtwFixture, campaignName))
	if err != nil {
		t.Fatalf("ParseMercenaryFile: %v", err)
	}
	regions, _ := parser.ParseRegionNames(parser.RegionsFilePath(rtwFixture, campaignName))

	repo := repository.NewMercenaryRepository(*file)
	svc := service.NewMercenaryService(repo, regions)

	// Untouched pool reports no change.
	if ct := svc.GetChangeType("Gaul_Pool"); ct != "none" {
		t.Fatalf("fresh pool change type = %q, want %q", ct, "none")
	}

	// Update: bump cost, add a region.
	pool, err := svc.GetPoolByName("Gaul_Pool")
	if err != nil {
		t.Fatalf("GetPoolByName: %v", err)
	}
	units := pool.Units
	units[0].Cost = 999
	if err := svc.UpdatePool("Gaul_Pool", append(pool.Regions, "Illyria"), units); err != nil {
		t.Fatalf("UpdatePool: %v", err)
	}
	if ct := svc.GetChangeType("Gaul_Pool"); ct != "modified" {
		t.Errorf("after update, change type = %q, want %q", ct, "modified")
	}
	updated, _ := svc.GetPoolByName("Gaul_Pool")
	if updated.Units[0].Cost != 999 {
		t.Errorf("updated cost = %d, want 999", updated.Units[0].Cost)
	}
	if len(updated.Regions) != 2 {
		t.Errorf("updated regions = %v, want 2 entries", updated.Regions)
	}

	// Revert restores the original.
	if err := svc.Revert("Gaul_Pool"); err != nil {
		t.Fatalf("Revert: %v", err)
	}
	reverted, _ := svc.GetPoolByName("Gaul_Pool")
	if reverted.Units[0].Cost == 999 {
		t.Errorf("cost still 999 after revert")
	}
	if ct := svc.GetChangeType("Gaul_Pool"); ct != "none" {
		t.Errorf("after revert, change type = %q, want %q", ct, "none")
	}

	// Create a new pool.
	if err := svc.CreatePool("New_Pool", []string{"Sardinia"}); err != nil {
		t.Fatalf("CreatePool: %v", err)
	}
	if ct := svc.GetChangeType("New_Pool"); ct != "added" {
		t.Errorf("new pool change type = %q, want %q", ct, "added")
	}

	// Rename it.
	if err := svc.Rename("New_Pool", "Renamed_Pool"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, err := svc.GetPoolByName("New_Pool"); err == nil {
		t.Errorf("old pool name still resolves after rename")
	}
	if _, err := svc.GetPoolByName("Renamed_Pool"); err != nil {
		t.Errorf("renamed pool not found: %v", err)
	}

	// Delete it.
	if err := svc.DeletePool("Renamed_Pool"); err != nil {
		t.Fatalf("DeletePool: %v", err)
	}
	if _, err := svc.GetPoolByName("Renamed_Pool"); err == nil {
		t.Errorf("deleted pool still resolves")
	}

	if !svc.HasUnsavedChanges() {
		t.Errorf("HasUnsavedChanges = false, want true")
	}

	svc.RevertAll()
	if svc.HasUnsavedChanges() {
		t.Errorf("HasUnsavedChanges = true after RevertAll")
	}
}

// ── Writer: save / apply / restore ──────────────────────────────────────────

func TestMercenaries_SaveApplyRestore_RTW(t *testing.T) {
	dataPath := setupModDir(t, rtwFixture)

	file, err := parser.ParseMercenaryFile(parser.MercenariesFilePath(dataPath, campaignName))
	if err != nil {
		t.Fatalf("ParseMercenaryFile: %v", err)
	}
	regions, _ := parser.ParseRegionNames(parser.RegionsFilePath(dataPath, campaignName))
	repo := repository.NewMercenaryRepository(*file)
	svc := service.NewMercenaryService(repo, regions)

	pool, err := svc.GetPoolByName("Sicily_Pool")
	if err != nil {
		t.Fatalf("GetPoolByName: %v", err)
	}
	units := pool.Units
	units[0].Cost = 12345
	if err := svc.UpdatePool("Sicily_Pool", pool.Regions, units); err != nil {
		t.Fatalf("UpdatePool: %v", err)
	}

	gw := writer.New(dataPath)
	if err := gw.SaveMercenariesDraft(campaignName, repo.GetData(), repo.GetChanges()); err != nil {
		t.Fatalf("SaveMercenariesDraft: %v", err)
	}

	// Game file must be untouched pre-Apply.
	preApply, err := parser.ParseMercenaryFile(parser.MercenariesFilePath(dataPath, campaignName))
	if err != nil {
		t.Fatalf("re-parse before apply: %v", err)
	}
	if p := findPool(t, preApply.Pools, "Sicily_Pool"); p.Units[0].Cost == 12345 {
		t.Fatalf("game file changed before Apply()")
	}

	if err := gw.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	postApply, err := parser.ParseMercenaryFile(parser.MercenariesFilePath(dataPath, campaignName))
	if err != nil {
		t.Fatalf("re-parse after apply: %v", err)
	}
	if p := findPool(t, postApply.Pools, "Sicily_Pool"); p.Units[0].Cost != 12345 {
		t.Fatalf("Apply did not write the change, got cost %d", p.Units[0].Cost)
	}
	// Untouched pool must be byte-identical to the original.
	if p := findPool(t, postApply.Pools, "Gaul_Pool"); p.Units[0].Cost != 400 {
		t.Fatalf("untouched Gaul_Pool changed: cost=%d", p.Units[0].Cost)
	}

	if err := gw.RestoreFromBackup(); err != nil {
		t.Fatalf("RestoreFromBackup: %v", err)
	}
	restored, err := parser.ParseMercenaryFile(parser.MercenariesFilePath(dataPath, campaignName))
	if err != nil {
		t.Fatalf("re-parse after restore: %v", err)
	}
	if p := findPool(t, restored.Pools, "Sicily_Pool"); p.Units[0].Cost != 500 {
		t.Fatalf("RestoreFromBackup did not revert the change, got cost %d, want 500", p.Units[0].Cost)
	}
}

// TestMercenaries_BaseGameFallback simulates a mod that doesn't ship its own
// campaign map: descr_mercenaries.txt/descr_regions.txt only exist in the base
// game's data folder. app.go's GetMercenaryCampaigns/LoadMercenaryCampaign fall
// back mod → base for exactly this case (see app.go's mercenaryRoots), built on
// top of these same parser primitives.
func TestMercenaries_BaseGameFallback(t *testing.T) {
	baseDataPath := setupModDir(t, rtwFixture) // acts as the "base game" data folder

	modDataPath := t.TempDir() + "/data"
	if err := os.MkdirAll(modDataPath, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", modDataPath, err)
	}

	// The mod's own data folder exists but has no campaign map at all.
	if got := parser.ListMercenaryCampaigns(modDataPath); len(got) != 0 {
		t.Fatalf("mod-only scan found campaigns %v, want none", got)
	}
	if got := parser.ListMercenaryCampaigns(baseDataPath); len(got) != 1 {
		t.Fatalf("base-game scan found %v, want 1 campaign", got)
	}
}
