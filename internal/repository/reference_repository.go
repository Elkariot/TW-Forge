package repository

func (r *InMemoryRepository) GetCultureNames() []string {
	return r.working.Cultures
}

func (r *InMemoryRepository) GetProjectileTypes() []string {
	return r.working.ProjectileTypes
}
