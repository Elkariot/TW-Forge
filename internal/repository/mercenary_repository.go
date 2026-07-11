package repository

import (
	"fmt"
	"slices"
	"tw-forge/internal/domain"
)

// MercenaryRepository manages one campaign's descr_mercenaries.txt. It is scoped to a
// single campaign at a time — app.go creates a new instance whenever the user switches
// the active campaign (see parser.ListMercenaryCampaigns). Like ProjectileRepository,
// it's a pure in-memory store keyed by pool name; persistence is orchestrated by the
// caller via GetData()/GetChanges() (fed into writer.GameWriter) and CommitSave().
type MercenaryRepository interface {
	GetAll() []domain.MercenaryPool
	GetByName(name string) (domain.MercenaryPool, bool)

	AddPool(p domain.MercenaryPool) error
	UpdatePool(originalName string, p domain.MercenaryPool) error
	DeletePool(name string) error

	RevertPool(name string) error
	RevertAll()

	HasUnsavedChanges() bool
	GetChanges() map[string]ChangeType
	GetData() domain.MercenaryFile
	CommitSave()
}

type InMemoryMercenaryRepository struct {
	original domain.MercenaryFile
	working  domain.MercenaryFile
	changes  map[string]ChangeType
}

func NewMercenaryRepository(data domain.MercenaryFile) *InMemoryMercenaryRepository {
	return &InMemoryMercenaryRepository{
		original: data,
		working:  data.DeepCopy(),
		changes:  make(map[string]ChangeType),
	}
}

func (r *InMemoryMercenaryRepository) GetAll() []domain.MercenaryPool {
	var out []domain.MercenaryPool
	for _, p := range r.working.Pools {
		if !p.IsDeleted {
			out = append(out, p)
		}
	}
	return out
}

func (r *InMemoryMercenaryRepository) GetByName(name string) (domain.MercenaryPool, bool) {
	for _, p := range r.working.Pools {
		if p.Name == name && !p.IsDeleted {
			return p, true
		}
	}
	return domain.MercenaryPool{}, false
}

func (r *InMemoryMercenaryRepository) indexByName(name string) int {
	for i, p := range r.working.Pools {
		if p.Name == name {
			return i
		}
	}
	return -1
}

func (r *InMemoryMercenaryRepository) AddPool(p domain.MercenaryPool) error {
	if _, ok := r.GetByName(p.Name); ok {
		return fmt.Errorf("pool %q already exists", p.Name)
	}
	r.working.Pools = append(r.working.Pools, p)
	r.changes[p.Name] = ChangeAdded
	return nil
}

func (r *InMemoryMercenaryRepository) UpdatePool(originalName string, p domain.MercenaryPool) error {
	i := r.indexByName(originalName)
	if i == -1 {
		return fmt.Errorf("pool %q not found", originalName)
	}
	r.working.Pools[i] = p
	if originalName != p.Name {
		r.changes[originalName] = ChangeDeleted
		r.changes[p.Name] = ChangeAdded
	} else if r.changes[originalName] != ChangeAdded {
		r.changes[originalName] = ChangeModified
	}
	return nil
}

func (r *InMemoryMercenaryRepository) DeletePool(name string) error {
	i := r.indexByName(name)
	if i == -1 {
		return fmt.Errorf("pool %q not found", name)
	}
	if r.changes[name] == ChangeAdded {
		r.working.Pools = slices.Delete(r.working.Pools, i, i+1)
		delete(r.changes, name)
		return nil
	}
	r.working.Pools[i].IsDeleted = true
	r.changes[name] = ChangeDeleted
	return nil
}

func (r *InMemoryMercenaryRepository) RevertPool(name string) error {
	changeType, changed := r.changes[name]
	if !changed {
		return nil
	}

	switch changeType {
	case ChangeAdded:
		if i := r.indexByName(name); i != -1 {
			r.working.Pools = slices.Delete(r.working.Pools, i, i+1)
		}
	case ChangeModified, ChangeDeleted:
		for _, orig := range r.original.Pools {
			if orig.Name == name {
				if i := r.indexByName(name); i != -1 {
					r.working.Pools[i] = orig
				} else {
					r.working.Pools = append(r.working.Pools, orig)
				}
				break
			}
		}
	}

	delete(r.changes, name)
	return nil
}

func (r *InMemoryMercenaryRepository) RevertAll() {
	r.working = r.original.DeepCopy()
	r.changes = make(map[string]ChangeType)
}

func (r *InMemoryMercenaryRepository) HasUnsavedChanges() bool {
	return len(r.changes) > 0
}

func (r *InMemoryMercenaryRepository) GetChanges() map[string]ChangeType {
	return r.changes
}

func (r *InMemoryMercenaryRepository) GetData() domain.MercenaryFile {
	return r.working
}

func (r *InMemoryMercenaryRepository) CommitSave() {
	r.original = r.working.DeepCopy()
	r.changes = make(map[string]ChangeType)
}
