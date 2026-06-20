package repository

import "tw-forge/internal/domain"

func (r *InMemoryRepository) GetFactions() []domain.Faction {
	return r.working.Factions
}

func (r *InMemoryRepository) GetFactionByName(name string) (domain.Faction, bool) {
	for _, f := range r.working.Factions {
		if f.Name == name {
			return f, true
		}
	}
	return domain.Faction{}, false
}

func (r *InMemoryRepository) GetFactionCulture(name string) (string, bool) {
	f, ok := r.GetFactionByName(name)
	return f.Culture, ok
}