package service

import (
	"fmt"
	"strings"
	"tw-forge/internal/domain"
	"tw-forge/internal/repository"
)

// ProjectileService edits descr_projectile.txt. It is game-version agnostic:
// RTW and M2TW share the same file format, so a single instance backed by the
// active gamePath serves both.
type ProjectileService struct {
	repo repository.ProjectileRepository
}

func NewProjectileService(repo repository.ProjectileRepository) *ProjectileService {
	return &ProjectileService{repo: repo}
}

func (s *ProjectileService) GetAll() []domain.Projectile {
	return s.repo.GetAll()
}

func (s *ProjectileService) GetDelays() []domain.ProjectileDelay {
	return s.repo.GetDelays()
}

func (s *ProjectileService) GetByName(name string) (*domain.Projectile, error) {
	p, ok := s.repo.GetByName(name)
	if !ok {
		return nil, fmt.Errorf("projectile not found: %s", name)
	}
	return &p, nil
}

func (s *ProjectileService) Update(originalName string, p domain.Projectile) error {
	if err := s.repo.UpdateProjectile(originalName, p); err != nil {
		return err
	}
	return s.repo.SaveDraft()
}

// Create clones templateName's block under newName. The new entry starts as an
// exact copy and can be edited afterwards via Update.
func (s *ProjectileService) Create(templateName, newName string) error {
	if _, exists := s.repo.GetByName(newName); exists {
		return fmt.Errorf("projectile %q already exists", newName)
	}
	tmpl, ok := s.repo.GetByName(templateName)
	if !ok {
		return fmt.Errorf("template %q not found", templateName)
	}

	tmpl.Name = newName
	tmpl.RawBlock = strings.Replace(tmpl.RawBlock, "projectile "+templateName, "projectile "+newName, 1)

	if err := s.repo.AddProjectile(tmpl); err != nil {
		return err
	}
	return s.repo.SaveDraft()
}

func (s *ProjectileService) Delete(name string) error {
	if err := s.repo.DeleteProjectile(name); err != nil {
		return err
	}
	return s.repo.SaveDraft()
}

func (s *ProjectileService) GetChangeType(name string) string {
	switch s.repo.GetChanges()[name] {
	case repository.ChangeModified:
		return "modified"
	case repository.ChangeAdded:
		return "added"
	case repository.ChangeDeleted:
		return "deleted"
	default:
		return "none"
	}
}

func (s *ProjectileService) Revert(name string) error {
	return s.repo.RevertProjectile(name)
}

func (s *ProjectileService) RevertAll() {
	s.repo.RevertAllProjectiles()
}

func (s *ProjectileService) HasUnsavedChanges() bool {
	return s.repo.HasUnsavedChanges()
}

func (s *ProjectileService) Save() error {
	return s.repo.Save()
}
