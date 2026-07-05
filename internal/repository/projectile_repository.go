package repository

import (
	"fmt"
	"slices"
	"tw-forge/internal/domain"
)

// ProjectileRepository manages descr_projectile.txt entries. It is game-version
// agnostic: RTW and M2TW share the exact same file format. Like GameRepository,
// it's a pure in-memory store — persistence is orchestrated by the caller via
// GetData()/GetChanges() (fed into writer.GameWriter) and CommitSave() (after
// writer.GameWriter.Apply() succeeds).
type ProjectileRepository interface {
	GetAll() []domain.Projectile
	GetDelays() []domain.ProjectileDelay
	GetByName(name string) (domain.Projectile, bool)

	AddProjectile(p domain.Projectile) error
	UpdateProjectile(originalName string, p domain.Projectile) error
	DeleteProjectile(name string) error

	RevertProjectile(name string) error
	RevertAllProjectiles()

	HasUnsavedChanges() bool
	GetChanges() map[string]ChangeType
	GetData() domain.ProjectileFile
	CommitSave()
}

type InMemoryProjectileRepository struct {
	original domain.ProjectileFile
	working  domain.ProjectileFile
	changes  map[string]ChangeType
}

func NewProjectile(data domain.ProjectileFile) *InMemoryProjectileRepository {
	return &InMemoryProjectileRepository{
		original: data,
		working:  data.DeepCopy(),
		changes:  make(map[string]ChangeType),
	}
}

func (r *InMemoryProjectileRepository) GetAll() []domain.Projectile {
	var out []domain.Projectile
	for _, p := range r.working.Projectiles {
		if !p.IsDeleted {
			out = append(out, p)
		}
	}
	return out
}

func (r *InMemoryProjectileRepository) GetDelays() []domain.ProjectileDelay {
	return r.working.Delays
}

func (r *InMemoryProjectileRepository) GetByName(name string) (domain.Projectile, bool) {
	for _, p := range r.working.Projectiles {
		if p.Name == name && !p.IsDeleted {
			return p, true
		}
	}
	return domain.Projectile{}, false
}

func (r *InMemoryProjectileRepository) indexByName(name string) int {
	for i, p := range r.working.Projectiles {
		if p.Name == name {
			return i
		}
	}
	return -1
}

func (r *InMemoryProjectileRepository) AddProjectile(p domain.Projectile) error {
	if _, ok := r.GetByName(p.Name); ok {
		return fmt.Errorf("projectile %q already exists", p.Name)
	}
	r.working.Projectiles = append(r.working.Projectiles, p)
	r.changes[p.Name] = ChangeAdded
	return nil
}

func (r *InMemoryProjectileRepository) UpdateProjectile(originalName string, p domain.Projectile) error {
	i := r.indexByName(originalName)
	if i == -1 {
		return fmt.Errorf("projectile %q not found", originalName)
	}
	r.working.Projectiles[i] = p
	if originalName != p.Name {
		r.changes[originalName] = ChangeDeleted
		r.changes[p.Name] = ChangeAdded
	} else if r.changes[originalName] != ChangeAdded {
		r.changes[originalName] = ChangeModified
	}
	return nil
}

func (r *InMemoryProjectileRepository) DeleteProjectile(name string) error {
	i := r.indexByName(name)
	if i == -1 {
		return fmt.Errorf("projectile %q not found", name)
	}
	if r.changes[name] == ChangeAdded {
		r.working.Projectiles = slices.Delete(r.working.Projectiles, i, i+1)
		delete(r.changes, name)
		return nil
	}
	r.working.Projectiles[i].IsDeleted = true
	r.changes[name] = ChangeDeleted
	return nil
}

func (r *InMemoryProjectileRepository) RevertProjectile(name string) error {
	changeType, changed := r.changes[name]
	if !changed {
		return nil
	}

	switch changeType {
	case ChangeAdded:
		if i := r.indexByName(name); i != -1 {
			r.working.Projectiles = slices.Delete(r.working.Projectiles, i, i+1)
		}
	case ChangeModified, ChangeDeleted:
		for _, orig := range r.original.Projectiles {
			if orig.Name == name {
				if i := r.indexByName(name); i != -1 {
					r.working.Projectiles[i] = orig
				} else {
					r.working.Projectiles = append(r.working.Projectiles, orig)
				}
				break
			}
		}
	}

	delete(r.changes, name)
	return nil
}

func (r *InMemoryProjectileRepository) RevertAllProjectiles() {
	r.working = r.original.DeepCopy()
	r.changes = make(map[string]ChangeType)
}

func (r *InMemoryProjectileRepository) HasUnsavedChanges() bool {
	return len(r.changes) > 0
}

func (r *InMemoryProjectileRepository) GetChanges() map[string]ChangeType {
	return r.changes
}

func (r *InMemoryProjectileRepository) GetData() domain.ProjectileFile {
	return r.working
}

func (r *InMemoryProjectileRepository) CommitSave() {
	r.original = r.working.DeepCopy()
	r.changes = make(map[string]ChangeType)
}
