package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var wwwDir string

// NewServer creates and returns the HTTP handler
func NewServer(www string) http.Handler {
	wwwDir = www
	mux := http.NewServeMux()
	registerRoutes(mux)
	return corsMiddleware(mux)
}

func registerRoutes(mux *http.ServeMux) {
	// Status
	mux.HandleFunc("/api/status", handleStatus)

	// Payloads
	mux.HandleFunc("/api/payloads", handlePayloads)
	mux.HandleFunc("/api/payloads/save", handlePayloadSave)
	mux.HandleFunc("/api/payloads/delete", handlePayloadDelete)

	// Categories & Tags
	mux.HandleFunc("/api/categories", handleCategories)
	mux.HandleFunc("/api/categories/save", handleCategorySave)
	mux.HandleFunc("/api/categories/delete", handleCategoryDelete)
	mux.HandleFunc("/api/tags", handleTags)

	// Inject
	mux.HandleFunc("/api/inject", handleInject)

	// WiFi
	mux.HandleFunc("/api/wifi", handleWifi)
	mux.HandleFunc("/api/wifi/save", handleWifiSave)
	mux.HandleFunc("/api/wifi/restart", handleWifiRestart)

	// Gadget
	mux.HandleFunc("/api/gadget", handleGadget)
	mux.HandleFunc("/api/gadget/save", handleGadgetSave)

	// Triggers
	mux.HandleFunc("/api/triggers", handleTriggers)
	mux.HandleFunc("/api/triggers/save", handleTriggersSave)
	mux.HandleFunc("/api/triggers/test", handleTriggersTest)

	// Library
	mux.HandleFunc("/api/library", handleLibrary)
	mux.HandleFunc("/api/library/import", handleLibraryImport)

	// Loot
	mux.HandleFunc("/api/loot", handleLoot)
	mux.HandleFunc("/api/loot/upload", handleLootUpload)
	mux.HandleFunc("/api/loot/download/", handleLootDownload)
	mux.HandleFunc("/api/loot/delete", handleLootDelete)
	mux.HandleFunc("/api/loot/clear", handleLootClear)

	// Logs
	mux.HandleFunc("/api/logs", handleLogs)
	mux.HandleFunc("/api/logs/stream", handleLogsStream)
	mux.HandleFunc("/api/logs/clear", handleLogsClear)

	// Static UI — catch-all
	mux.HandleFunc("/", handleStatic)
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(wwwDir, r.URL.Path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// SPA fallback — serve index.html for all unknown routes
		http.ServeFile(w, r, filepath.Join(wwwDir, "index.html"))
		return
	}
	http.ServeFile(w, r, path)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
