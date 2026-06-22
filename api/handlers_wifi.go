package api

import (
	"net/http"

	"github.com/pwnduck/core"
	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

func handleWifi(w http.ResponseWriter, r *http.Request) {
	cfg := store.GetWifi()
	jsonOK(w, map[string]any{
		"ssid":     cfg.SSID,
		"password": cfg.Password,
		"channel":  cfg.Channel,
		"auth":     cfg.Auth,
		"hidden":   cfg.Hidden,
		"status":   core.WifiRunning(),
	})
}

func handleWifiSave(w http.ResponseWriter, r *http.Request) {
	var cfg store.WifiConfig
	if err := decode(r, &cfg); err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	if err := core.ApplyWifi(cfg); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
}

func handleWifiRestart(w http.ResponseWriter, r *http.Request) {
	if err := core.RestartWifi(); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	logger.Info(logger.SrcWifi, "WiFi restarted via API")
	jsonOK(w, map[string]bool{"ok": true})
}
