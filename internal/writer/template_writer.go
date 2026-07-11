package writer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
	"tw-forge/internal/domain"
)

// templateSnapshotFiles lists the same relative paths RestoreFromBackup covers, so a
// template capture/switch can undo or redo exactly what "Restore Original" can — icon and
// texture files under UI/ui are intentionally excluded, same as backup/restore (see
// RestoreFromBackup's own comment: those are never safety-netted).
func (w *GameWriter) templateSnapshotFiles() []string {
	files := append([]string(nil), managedFiles...)
	return append(files, w.modelDBRelPath(), "descr_model_battle.txt")
}

func (w *GameWriter) templateDir(name string) string {
	return filepath.Join(w.templatesPath, name)
}

// validTemplateName rejects empty names and anything that isn't a plain, single path
// segment — template names become directory names under templatesPath, so path
// separators or ".."/"." must not be allowed to escape it.
func validTemplateName(name string) error {
	if name == "" {
		return fmt.Errorf("template name required")
	}
	if name != filepath.Base(name) || name == "." || name == ".." {
		return fmt.Errorf("invalid template name %q", name)
	}
	return nil
}

// ListTemplates returns every saved template, sorted by name.
func (w *GameWriter) ListTemplates() ([]domain.TemplateInfo, error) {
	entries, err := os.ReadDir(w.templatesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []domain.TemplateInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		var created time.Time
		if info, err := e.Info(); err == nil {
			created = info.ModTime()
		}
		out = append(out, domain.TemplateInfo{Name: e.Name(), CreatedAt: created})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// CreateTemplate snapshots the current game files (the same set RestoreFromBackup can
// undo) into a newly named template directory, and makes it the current template —
// like "git checkout -b": from now on, every Apply() folds its changes straight into
// this template (see syncCurrentTemplate) until the user switches to another one or to
// "no template" (ClearTemplate).
func (w *GameWriter) CreateTemplate(name string) error {
	if err := w.ensureInit(); err != nil {
		return err
	}
	if err := validTemplateName(name); err != nil {
		return err
	}
	dir := w.templateDir(name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("template %q already exists", name)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create template dir: %w", err)
	}
	if err := w.snapshotGamePathInto(dir); err != nil {
		return err
	}
	return w.saveTemplateState(domain.TemplateState{Current: name})
}

// snapshotGamePathInto copies the current game files (the same set RestoreFromBackup
// can undo) into dir, overwriting whatever is already there.
func (w *GameWriter) snapshotGamePathInto(dir string) error {
	for _, rel := range w.templateSnapshotFiles() {
		src := filepath.Join(w.gamePath, rel)
		if _, err := os.Stat(src); err != nil {
			continue // file not present in this game/mod
		}
		dst := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("mkdir template dir for %s: %w", rel, err)
		}
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("snapshot %s: %w", rel, err)
		}
	}
	campaignRel := filepath.Join("world", "maps", "campaign")
	if err := copyTree(filepath.Join(w.gamePath, campaignRel), filepath.Join(dir, campaignRel)); err != nil {
		return fmt.Errorf("snapshot mercenaries: %w", err)
	}
	return nil
}

// LoadTemplate overwrites the current game files with a previously saved template's
// files, then clears the draft directory — same reasoning as RestoreFromBackup: a stale
// draft must not be silently re-applied afterwards. Callers must reload all in-memory
// game state after this succeeds (see App.SwitchTemplate, which re-runs InitGame).
func (w *GameWriter) LoadTemplate(name string) error {
	if err := validTemplateName(name); err != nil {
		return err
	}
	dir := w.templateDir(name)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("template %q not found", name)
	}
	for _, rel := range w.templateSnapshotFiles() {
		src := filepath.Join(dir, rel)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := copyFile(src, filepath.Join(w.gamePath, rel)); err != nil {
			return fmt.Errorf("restore %s: %w", rel, err)
		}
	}
	campaignRel := filepath.Join("world", "maps", "campaign")
	if err := copyTree(filepath.Join(dir, campaignRel), filepath.Join(w.gamePath, campaignRel)); err != nil {
		return fmt.Errorf("restore mercenaries: %w", err)
	}
	if err := os.RemoveAll(w.draftPath); err != nil {
		return err
	}
	return w.saveTemplateState(domain.TemplateState{Current: name})
}

// DeleteTemplate removes a saved template. If it was the currently selected one, the
// selection is cleared (it no longer refers to anything).
func (w *GameWriter) DeleteTemplate(name string) error {
	if err := validTemplateName(name); err != nil {
		return err
	}
	dir := w.templateDir(name)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("template %q not found", name)
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	st := w.loadTemplateState()
	if st.Current == name {
		return w.saveTemplateState(domain.TemplateState{})
	}
	return nil
}

// ClearTemplate detaches from the current template without touching any game files —
// used when the user picks "no template" from the templates dropdown. It's always
// non-destructive: nothing on disk changes, so callers don't need to reload game state
// afterwards. Future edits/applies are no longer folded into any template until a new
// one is created or loaded.
func (w *GameWriter) ClearTemplate() error {
	return w.saveTemplateState(domain.TemplateState{})
}

// GetTemplateState returns the currently selected template (empty if none) and whether
// the game files have drifted from it since it was last captured or loaded.
func (w *GameWriter) GetTemplateState() domain.TemplateState {
	return w.loadTemplateState()
}

func (w *GameWriter) loadTemplateState() domain.TemplateState {
	data, err := os.ReadFile(w.templateStatePath)
	if err != nil {
		return domain.TemplateState{}
	}
	var st domain.TemplateState
	_ = json.Unmarshal(data, &st)
	return st
}

func (w *GameWriter) saveTemplateState(st domain.TemplateState) error {
	if err := os.MkdirAll(filepath.Dir(w.templateStatePath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(w.templateStatePath, data, 0644)
}

// syncCurrentTemplate is called after Apply() actually writes something to disk. If a
// template is currently active, its snapshot is silently re-captured from the game
// files so the new changes are folded straight into it — matching the "I'm inside a
// template, everything I do belongs to it" mental model, with no repeated "save as
// template?" prompt on every subsequent apply. With no active template, there's
// nothing to fold into, so the change is just flagged as untemplated.
func (w *GameWriter) syncCurrentTemplate() error {
	st := w.loadTemplateState()
	if st.Current == "" {
		return w.saveTemplateState(domain.TemplateState{Dirty: true})
	}
	if err := w.snapshotGamePathInto(w.templateDir(st.Current)); err != nil {
		return err
	}
	return w.saveTemplateState(domain.TemplateState{Current: st.Current})
}

// copyTree copies every file under src to the matching relative path under dst. No-op if
// src doesn't exist.
func copyTree(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if mkErr := os.MkdirAll(filepath.Dir(target), 0755); mkErr != nil {
			return mkErr
		}
		return copyFile(path, target)
	})
}
