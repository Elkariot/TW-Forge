package domain

type Faction struct {
	Name            string
	DisplayName     string
	Culture         string
	PrimaryColour   Colour
	SecondaryColour Colour
}

type Colour struct {
	R, G, B uint8
}
