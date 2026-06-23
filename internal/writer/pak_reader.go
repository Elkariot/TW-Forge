package writer

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// extractFromPaks ищет файл по wantFileName в faction-специфичной папке во всех .pak архивах.
// Сначала ищет в модовых паках (dataDir/packs/), потом в базовых паках игры.
// wantFileName уже должен быть в верхнем регистре, например "#WARBAND_HUNTSMAN_GAUL.TGA"
// или "WARBAND_HUNTSMAN_GAUL_INFO.TGA".
func extractFromPaks(dataDir, wantFileName, faction string) []byte {
	wantFaction := strings.ToUpper(faction)

	// Модовые паки в первую очередь (кастомные портреты, оверрайды)
	if data := searchPaksDir(filepath.Join(dataDir, "packs"), wantFileName, wantFaction); data != nil {
		return data
	}
	// Базовые паки игры (RTW vanilla)
	if baseDir := findBasePacksDir(dataDir); baseDir != "" {
		return searchPaksDir(baseDir, wantFileName, wantFaction)
	}
	return nil
}

// findBasePacksDir поднимается по дереву директорий от modDataDir и ищет соседний data/packs/.
// Для RTW ME мода: ME/Data → RTW root/data/packs.
func findBasePacksDir(modDataDir string) string {
	dir := filepath.Dir(modDataDir)
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		candidate := filepath.Join(parent, "data", "packs")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = parent
	}
}

func searchPaksDir(packsDir, wantFileName, wantFaction string) []byte {
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".pak") {
			continue
		}
		if data := readFromPak(filepath.Join(packsDir, e.Name()), wantFileName, wantFaction); data != nil {
			return data
		}
	}
	return nil
}

// readFromPak открывает RTW PAK0 архив и извлекает файл с именем wantFileName.
// Сначала ищет faction-специфичный путь, fallback — любая папка фракции.
func readFromPak(pakPath, wantFileName, wantFaction string) []byte {
	f, err := os.Open(pakPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	hdr := make([]byte, 12)
	if _, err := f.Read(hdr); err != nil || string(hdr[:4]) != "PAK0" {
		return nil
	}
	pathTableWords := binary.LittleEndian.Uint32(hdr[4:8])
	numFiles := binary.LittleEndian.Uint32(hdr[8:12])

	pathTable := make([]byte, pathTableWords*2)
	if _, err := f.Read(pathTable); err != nil {
		return nil
	}
	offsetTable := make([]byte, numFiles*4)
	if _, err := f.Read(offsetTable); err != nil {
		return nil
	}

	factionSuffix := `\` + wantFaction + `\` + wantFileName
	anySuffix := `\` + wantFileName
	var fallbackIdx int32 = -1

	pos := 0
	for i := uint32(0); i < numFiles; i++ {
		path := readUTF16LEString(pathTable, &pos)
		upper := strings.ToUpper(path)
		if strings.HasSuffix(upper, factionSuffix) {
			return extractPakFile(f, offsetTable, i, numFiles)
		}
		if fallbackIdx < 0 && strings.HasSuffix(upper, anySuffix) {
			fallbackIdx = int32(i)
		}
	}
	if fallbackIdx >= 0 {
		return extractPakFile(f, offsetTable, uint32(fallbackIdx), numFiles)
	}
	return nil
}

func readUTF16LEString(table []byte, pos *int) string {
	start := *pos
	for *pos+1 < len(table) && !(table[*pos] == 0 && table[*pos+1] == 0) {
		*pos += 2
	}
	words := make([]uint16, (*pos-start)/2)
	for j := range words {
		words[j] = binary.LittleEndian.Uint16(table[start+j*2:])
	}
	*pos += 2
	return string(utf16.Decode(words))
}

func extractPakFile(f *os.File, offsetTable []byte, idx, numFiles uint32) []byte {
	offset := binary.LittleEndian.Uint32(offsetTable[idx*4:])
	var size uint32
	if idx+1 < numFiles {
		size = binary.LittleEndian.Uint32(offsetTable[(idx+1)*4:]) - offset
	} else {
		stat, err := f.Stat()
		if err != nil {
			return nil
		}
		size = uint32(stat.Size()) - offset
	}
	data := make([]byte, size)
	if _, err := f.ReadAt(data, int64(offset)); err != nil {
		return nil
	}
	return data
}
