package tests

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// rtwFixture and m2twFixture point at the committed, hand-trimmed mod fixtures
// (see tests/testdata/README.md) used by every test in this package.
const (
	rtwFixture  = "testdata/rtw/data"
	m2twFixture = "testdata/m2tw/data"
)

// rtwCampaign and m2twCampaign name the campaign folder inside each fixture's
// world/maps/campaign/ that carries the mercenaries + regions test data.
const campaignName = "imperial_campaign"

// setupModDir copies a fixture's data folder into a fresh temp directory and
// returns the path to the copy's "data" folder, so GameWriter's backup/draft
// staging (created as a sibling "_modding_editor" folder) never touches the
// repo's committed fixtures and is cleaned up automatically with the temp dir.
func setupModDir(t *testing.T, fixture string) string {
	t.Helper()

	root := t.TempDir()
	dst := filepath.Join(root, "data")
	if err := copyDir(fixture, dst); err != nil {
		t.Fatalf("copy fixture %s: %v", fixture, err)
	}
	return dst
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
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
