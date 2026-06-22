package api

import (
	"net/http"

	"github.com/pwnduck/core"
	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

func handleTriggers(w http.ResponseWriter, r *http.Request) {
	cfg := store.GetTriggerConfig()
	jsonOK(w, map[string]any{
		"triggers": cfg.Triggers,
		"enabled":  cfg.Enabled,
	})
}

func handleTriggersSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Triggers []store.Trigger `json:"triggers"`
		Enabled  bool            `json:"enabled"`
	}
	if err := decode(r, &req); err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	if req.Triggers == nil {
		req.Triggers = []store.Trigger{}
	}
	cfg := store.TriggerConfig{
		Enabled:  req.Enabled,
		Triggers: req.Triggers,
	}
	if err := store.SetTriggerConfig(cfg); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	logger.Info(logger.SrcTrigger, "Triggers updated")
	jsonOK(w, map[string]bool{"ok": true})
}

func handleTriggersTest(w http.ResponseWriter, r *http.Request) {
	if err := core.TestTriggers(); err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
}
