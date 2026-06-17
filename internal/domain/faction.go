package domain

type Faction struct {
	Name            string
	Culture         string
	PrimaryColour   Colour
	SecondaryColour Colour
}

type Colour struct {
	R, G, B uint8
}
