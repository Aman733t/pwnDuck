package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type LootFile struct {
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

var lootDir string

func InitLoot(dir string) error {
	lootDir = dir
	return os.MkdirAll(dir, 0755)
}

func ListLoot() []LootFile {
	entries, err := os.ReadDir(lootDir)
	if err != nil {
		return []LootFile{}
	}
	files := make([]LootFile, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, LootFile{
			Filename:  e.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format(time.RFC3339),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAt > files[j].CreatedAt
	})
	return files
}

func SaveLoot(filename string, data []byte) (string, error) {
	safe := sanitizeFilename(filename)
	ts := time.Now().Format("20060102_150405")
	name := fmt.Sprintf("%s_%s", ts, safe)
	path := filepath.Join(lootDir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return name, nil
}

func GetLootPath(filename string) string {
	return filepath.Join(lootDir, sanitizeFilename(filename))
}

func DeleteLoot(filename string) error {
	return os.Remove(filepath.Join(lootDir, sanitizeFilename(filename)))
}

func ClearLoot() (int, error) {
	entries, err := os.ReadDir(lootDir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			if err := os.Remove(filepath.Join(lootDir, e.Name())); err == nil {
				count++
			}
		}
	}
	return count, nil
}

func sanitizeFilename(name string) string {
	var b strings.Builder
	for _, c := range name {
		if c == '/' || c == '\\' || c == '\'' || c == '"' {
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}
