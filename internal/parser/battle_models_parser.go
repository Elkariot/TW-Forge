package parser

import (
	"tw-forge/internal/domain"
	"os"
	"strconv"
	"strings"
)

// ParseBattleModels parses battle_models.modeldb into an indexed structure.
// The file uses Boost serialization text format; we don't parse the full schema —
// instead we find entry boundaries and keep raw text blocks for safe round-trips.
func ParseBattleModels(path string) (*domain.BattleModelsDB, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(content, "\n")

	db := &domain.BattleModelsDB{
		Index: make(map[string]int),
	}

	if len(lines) == 0 {
		return db, nil
	}

	// First line is the Boost archive header: "22 serialization::archive 3 0 0 0 0 TOTAL 0 0"
	db.Header = strings.TrimSpace(lines[0])
	db.Count = extractModelDBCount(db.Header)

	// Find line indices where model entries begin using a two-step heuristic:
	//   1. Line is exactly "N name" (2 tokens, N = len(name), name has no '/' or '.')
	//   2. The NEXT non-empty line's first token is either a float (has '.') or a small int ≤ 2
	//      (model scales are 0.5–2.0; texture path lengths start at 40+)
	type entryStart struct {
		lineIdx int
		name    string
	}
	var starts []entryStart

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil || n <= 0 || n > 100 {
			continue
		}
		if len(parts[1]) != n {
			continue
		}
		if strings.ContainsAny(parts[1], "/.") {
			continue
		}
		// Look at the next non-empty line
		for j := i + 1; j < len(lines); j++ {
			next := strings.TrimSpace(lines[j])
			if next == "" {
				continue
			}
			first := strings.Fields(next)[0]
			if strings.Contains(first, ".") {
				// Float → model scale
				if _, e := strconv.ParseFloat(first, 64); e == nil {
					starts = append(starts, entryStart{i, parts[1]})
				}
			} else {
				v, e := strconv.Atoi(first)
				if e == nil && v <= 2 {
					// Small int: scale=0/1/2 or Boost tracking zero
					starts = append(starts, entryStart{i, parts[1]})
				}
			}
			break
		}
	}

	// Build raw block for each entry (from its start line to the line before the next entry)
	for i, s := range starts {
		endLine := len(lines)
		if i+1 < len(starts) {
			endLine = starts[i+1].lineIdx
		}
		block := strings.Join(lines[s.lineIdx:endLine], "\n")
		model := domain.BattleModel{
			Name:     s.name,
			RawBlock: block,
		}
		db.Index[s.name] = len(db.Models)
		db.Models = append(db.Models, model)
	}

	return db, nil
}

// extractModelDBCount pulls the total entry count from the header line.
// Header format: "22 serialization::archive 3 0 0 0 0 <count> 0 0"
func extractModelDBCount(header string) int {
	fields := strings.Fields(header)
	// After the length-prefixed string "serialization::archive" (indices 0 and 1),
	// come: version(2), 4 flags(3-6), count(7), 2 trailing(8-9)
	if len(fields) >= 8 {
		v, err := strconv.Atoi(fields[7])
		if err == nil {
			return v
		}
	}
	return 0
}
