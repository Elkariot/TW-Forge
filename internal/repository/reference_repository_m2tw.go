package repository

func (r *InMemoryM2TWRepository) GetReligions() []string {
	return r.working.Religions
}

func (r *InMemoryM2TWRepository) GetHiddenResources() []string {
	return r.working.HiddenResources
}

func (r *InMemoryM2TWRepository) GetCultureNames() []string {
	return r.working.Cultures
}

func (r *InMemoryM2TWRepository) GetProjectileTypes() []string {
	return r.working.ProjectileTypes
}
