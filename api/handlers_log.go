package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pwnduck/logger"
)

func handleLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}
	level := r.URL.Query().Get("level")
	source := r.URL.Query().Get("source")
	jsonOK(w, logger.Get(limit, level, source))
}

func handleLogsStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	ch := logger.Subscribe()
	defer logger.Unsubscribe(ch)

	// Send a ping immediately so client knows connection is alive
	fmt.Fprintf(w, ": ping\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func handleLogsClear(w http.ResponseWriter, r *http.Request) {
	logger.Clear()
	logger.Info(logger.SrcSystem, "Logs cleared")
	jsonOK(w, map[string]bool{"ok": true})
}
