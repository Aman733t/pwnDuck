package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

var libraryDir string

func SetLibraryDir(dir string) {
	libraryDir = dir
}

type LibraryEntry struct {
	store.Payload
	Path string `json:"path"`
}

func handleLibrary(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	results := make([]LibraryEntry, 0)

	filepath.Walk(libraryDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Base(path) != "payload.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var p store.Payload
		if err := json.Unmarshal(data, &p); err != nil {
			return nil
		}
		if category != "" && !strings.EqualFold(p.Category, category) {
			return nil
		}
		rel, _ := filepath.Rel(libraryDir, filepath.Dir(path))
		results = append(results, LibraryEntry{Payload: p, Path: rel})
		return nil
	})

	jsonOK(w, results)
}

func handleLibraryImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decode(r, &req); err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	data, err := os.ReadFile(filepath.Join(libraryDir, req.Path, "payload.json"))
	if err != nil {
		jsonErr(w, "payload not found", 404)
		return
	}

	var p store.Payload
	if err := json.Unmarshal(data, &p); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Reset ID so it gets a new one on save
	p.ID = ""
	if err := store.SavePayload(&p); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logger.Info(logger.SrcSystem, fmt.Sprintf("Imported from library: %s", p.Name))
	jsonOK(w, map[string]any{"ok": true, "id": p.ID})
}
