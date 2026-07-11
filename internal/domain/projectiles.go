package domain

// Projectile represents one "projectile <name> ... " entry from descr_projectile.txt.
// The file format is identical between RTW and M2TW and is far less regular than EDU
// (many optional keywords, flags, comments, per-weapon particle-effect blocks). Only the
// ballistics-relevant fields are parsed into typed values for editing; RawBlock keeps the
// original text so saving patches just the edited lines in place, leaving effects, display
// flags, models and comments untouched.
type Projectile struct {
	Name string

	FlamingOf   string // set if this entry is a "flaming <base>" variant of another projectile
	ExplodingOf string // set if this entry is an "exploding <base>" variant of another projectile

	Damage         int
	DamageToTroops *int // siege ammo only: separate anti-personnel damage

	Radius float64
	Mass   float64

	Area *float64 // splash radius; present only on area_effect projectiles

	AccuracyVsUnits     *float64
	AccuracyVsBuildings *float64
	AccuracyVsTowers    *float64

	MinAngle int
	MaxAngle int
	Velocity []float64 // one value = fixed speed, two values = randomized range

	// Effect is the flight/trail effect name (the "model" of the projectile as it
	// travels — the mesh, if any, is bundled inside this effect's own definition,
	// not chosen separately). The six End* fields are impact-effect names for the
	// different things a projectile can hit; empty string means the line is absent.
	Effect                  string
	EndEffect               string
	EndManEffect            string
	EndPackageEffect        string
	EndShatterEffect        string
	EndShatterManEffect     string
	EndShatterPackageEffect string

	Flags []string // bare boolean keywords present in the block: fiery, affected_by_rain, ground_shatter, body_piercing, grapeshot, prefer_high, no_ae_on_ram, effect_only, cow_carcass...

	IsDeleted bool

	RawBlock string // original block text (from "projectile <name>" line up to, but excluding, the blank line before the next entry)
}

// ProjectileDelay is a top-of-file "delay <type> <seconds>" directive, global to all
// projectiles of that type (standard/flaming/gunpowder). Read-only reference data.
type ProjectileDelay struct {
	Type    string
	Seconds float64
}

// ManagedProjectileFlags are the bare (no-value) boolean keywords in
// descr_projectile.txt that the parser/writer recognize as editable toggles.
// Any other bare keyword is preserved verbatim in RawBlock but never parsed
// into Flags or touched on save.
var ManagedProjectileFlags = []string{
	"fiery",
	"affected_by_rain",
	"ground_shatter",
	"body_piercing",
	"grapeshot",
	"prefer_high",
	"no_ae_on_ram",
	"effect_only",
	"cow_carcass",
}

// ProjectileFile is the parsed content of descr_projectile.txt.
type ProjectileFile struct {
	Delays      []ProjectileDelay
	Projectiles []Projectile
}

func (f ProjectileFile) DeepCopy() ProjectileFile {
	delays := make([]ProjectileDelay, len(f.Delays))
	copy(delays, f.Delays)

	projectiles := make([]Projectile, len(f.Projectiles))
	for i, p := range f.Projectiles {
		projectiles[i] = copyProjectile(p)
	}

	return ProjectileFile{Delays: delays, Projectiles: projectiles}
}

func copyProjectile(p Projectile) Projectile {
	if p.DamageToTroops != nil {
		v := *p.DamageToTroops
		p.DamageToTroops = &v
	}
	if p.Area != nil {
		v := *p.Area
		p.Area = &v
	}
	if p.AccuracyVsUnits != nil {
		v := *p.AccuracyVsUnits
		p.AccuracyVsUnits = &v
	}
	if p.AccuracyVsBuildings != nil {
		v := *p.AccuracyVsBuildings
		p.AccuracyVsBuildings = &v
	}
	if p.AccuracyVsTowers != nil {
		v := *p.AccuracyVsTowers
		p.AccuracyVsTowers = &v
	}
	p.Velocity = copyFloats(p.Velocity)
	p.Flags = copyStrings(p.Flags)
	return p
}

func copyFloats(s []float64) []float64 {
	if s == nil {
		return nil
	}
	c := make([]float64, len(s))
	copy(c, s)
	return c
}
