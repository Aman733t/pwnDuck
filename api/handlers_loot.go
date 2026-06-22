package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

func handleLoot(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, store.ListLoot())
}

func handleLootUpload(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		filename = fmt.Sprintf("loot_%d", r.ContentLength)
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	name, err := store.SaveLoot(filename, data)
	if err != nil {
		logger.Error(logger.SrcExfil, "Upload failed: "+err.Error())
		jsonErr(w, err.Error(), 500)
		return
	}
	logger.Success(logger.SrcExfil, fmt.Sprintf("Received: %s (%d bytes)", name, len(data)))
	jsonOK(w, map[string]any{"ok": true, "filename": name, "size": len(data)})
}

func handleLootDownload(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/loot/download/")
	path := store.GetLootPath(filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "not found", 404)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	http.ServeFile(w, r, path)
}

func handleLootDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename string `json:"filename"`
	}
	decode(r, &req)
	store.DeleteLoot(req.Filename)
	logger.Warn(logger.SrcExfil, "Loot deleted: "+req.Filename)
	jsonOK(w, map[string]bool{"ok": true})
}

func handleLootClear(w http.ResponseWriter, r *http.Request) {
	count, _ := store.ClearLoot()
	logger.Warn(logger.SrcExfil, fmt.Sprintf("Loot cleared: %d files", count))
	jsonOK(w, map[string]any{"ok": true, "deleted": count})
}
