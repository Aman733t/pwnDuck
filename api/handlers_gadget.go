package api

import (
	"net/http"

	"github.com/pwnduck/core"
	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

func handleGadget(w http.ResponseWriter, r *http.Request) {
	cfg := store.GetGadget()
	jsonOK(w, map[string]any{
		"hid":          cfg.HID,
		"ethernet":     cfg.Ethernet,
		"mass_storage": cfg.MassStorage,
		"vendor_id":    cfg.VendorID,
		"product_id":   cfg.ProductID,
		"manufacturer": cfg.Manufacturer,
		"product":      cfg.Product,
		"available":    core.GadgetAvailable(),
	})
}

func handleGadgetSave(w http.ResponseWriter, r *http.Request) {
	var cfg store.GadgetConfig
	if err := decode(r, &cfg); err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	if err := store.SetGadget(cfg); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	// Re-apply gadget with new config
	if core.GadgetAvailable() {
		if err := core.SetupGadget(); err != nil {
			logger.Error(logger.SrcGadget, "Gadget re-setup failed: "+err.Error())
			jsonErr(w, err.Error(), 500)
			return
		}
	}
	logger.Info(logger.SrcGadget, "Gadget config updated")
	jsonOK(w, map[string]bool{"ok": true})
}
