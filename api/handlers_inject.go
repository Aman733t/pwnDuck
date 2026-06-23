package api

import (
	"net/http"

	"github.com/pwnduck/core"
	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

func handleInject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Payload string `json:"payload"`
		Name    string `json:"name"`
	}
	if err := decode(r, &req); err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	if req.Payload == "" {
		jsonErr(w, "payload is empty", 400)
		return
	}
	if err := core.RunDucky(req.Payload); err != nil {
		logger.Error(logger.SrcHID, "Inject failed: "+err.Error())
		jsonErr(w, err.Error(), 500)
		return
	}
	name := req.Name
	if name == "" {
		name = "manual"
	}
	logger.Success(logger.SrcHID, "Injected: "+name)
	jsonOK(w, map[string]bool{"ok": true})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	cfg := store.GetTriggerConfig()
	gadget := store.GetGadget()
	jsonOK(w, map[string]any{
		"hid":             core.HIDAvailable(),
		"wifi":            core.WifiRunning(),
		"ethernet":        gadget.Ethernet,
		"mass_storage":    gadget.MassStorage,
		"udc_state":       core.ReadUDCState(),
		"eth_client_ip":   core.EthernetClientIP(),
		"trigger_enabled": cfg.Enabled,
		"trigger_count":   len(cfg.Triggers),
		"payload_count":   len(store.GetAllPayloads()),
		"loot_count":      len(store.ListLoot()),
	})
}

func handleInjectCancel(w http.ResponseWriter, r *http.Request) {
	if !core.IsInjecting() {
		jsonErr(w, "no injection in progress", 400)
		return
	}
	core.CancelInject()
	logger.Warn(logger.SrcHID, "Injection cancelled by user")
	jsonOK(w, map[string]bool{"ok": true})
}