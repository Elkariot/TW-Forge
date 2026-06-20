package repository

import "tw-forge/internal/domain"

func (r *InMemoryM2TWRepository) GetFactions() []domain.M2TWFaction {
	return r.working.Factions
}

func (r *InMemoryM2TWRepository) GetFactionByName(name string) (domain.M2TWFaction, bool) {
	for _, f := range r.working.Factions {
		if f.Name == name {
			return f, true
		}
	}
	return domain.M2TWFaction{}, false
}
