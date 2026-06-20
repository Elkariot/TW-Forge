package domain

// BattleModel represents one entry in battle_models.modeldb.
// The file uses Boost serialization text format, so we keep raw blocks
// for safe round-trip: parse names only, preserve raw bytes for writing.
type BattleModel struct {
	Name     string
	RawBlock string // the original text block (including name line), used for copy-based operations
}

// BattleModelsDB is the parsed modeldb file.
type BattleModelsDB struct {
	Header string                  // first line verbatim ("22 serialization::archive 3 ...")
	Count  int                     // total model count from header
	Models []BattleModel           // all entries
	Index  map[string]int          // name → index in Models
}
