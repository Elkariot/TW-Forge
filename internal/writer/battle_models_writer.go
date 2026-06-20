package writer

import (
	"fmt"
	"modding-utils/internal/domain"
	"os"
	"strconv"
	"strings"
)

// CopyModelEntry copies the entry named srcName to a new entry named dstName
// in the modeldb. If dstName already exists, it is a no-op.
// Returns the modified BattleModelsDB (the original is not mutated).
func CopyModelEntry(db *domain.BattleModelsDB, srcName, dstName string) (*domain.BattleModelsDB, error) {
	if _, exists := db.Index[dstName]; exists {
		return db, nil // already exists
	}
	srcIdx, ok := db.Index[srcName]
	if !ok {
		return nil, fmt.Errorf("model %q not found in battle_models.modeldb", srcName)
	}

	srcBlock := db.Models[srcIdx].RawBlock

	// Replace the name in the first line: "N srcName\n" → "N dstName\n"
	firstNL := strings.Index(srcBlock, "\n")
	if firstNL == -1 {
		firstNL = len(srcBlock)
	}
	newNameLine := fmt.Sprintf("%d %s", len(dstName), dstName)
	newBlock := newNameLine + srcBlock[firstNL:]

	newDB := &domain.BattleModelsDB{
		Header: db.Header,
		Count:  db.Count + 1,
		Models: make([]domain.BattleModel, len(db.Models)+1),
		Index:  make(map[string]int, len(db.Index)+1),
	}
	copy(newDB.Models, db.Models)
	newDB.Models[len(db.Models)] = domain.BattleModel{
		Name:     dstName,
		RawBlock: newBlock,
	}
	for k, v := range db.Index {
		newDB.Index[k] = v
	}
	newDB.Index[dstName] = len(db.Models)

	return newDB, nil
}

// WriteBattleModels writes the BattleModelsDB to a file, updating the count in the header.
func WriteBattleModels(db *domain.BattleModelsDB, path string) error {
	header := updateModelDBCount(db.Header, db.Count)

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n")

	for _, m := range db.Models {
		sb.WriteString(m.RawBlock)
		if !strings.HasSuffix(m.RawBlock, "\n") {
			sb.WriteString("\n")
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// updateModelDBCount replaces the count value in the header line.
// Header: "22 serialization::archive 3 0 0 0 0 <count> 0 0"
func updateModelDBCount(header string, newCount int) string {
	fields := strings.Fields(header)
	if len(fields) < 8 {
		return header
	}
	fields[7] = strconv.Itoa(newCount)
	return strings.Join(fields, " ")
}
