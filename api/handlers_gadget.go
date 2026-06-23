package api

import (
	"net/http"
	"os"
	"path/filepath"

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

// handleEthernetStatus returns client IP and detected OS
func handleEthernetStatus(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, core.EthernetStatus())
}

// handleUMSFiles lists files on the UMS image
func handleUMSFiles(w http.ResponseWriter, r *http.Request) {
	mountPoint := "/tmp/pwnduck-ums"

	if err := core.MountUMSImage(mountPoint); err != nil {
		jsonErr(w, "mount failed: "+err.Error(), 500)
		return
	}
	defer core.UnmountUMSImage(mountPoint)

	entries, err := os.ReadDir(mountPoint)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	files := make([]map[string]any, 0)
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		files = append(files, map[string]any{
			"name":  e.Name(),
			"is_dir": e.IsDir(),
			"size":  size,
			"path":  filepath.Join("/", e.Name()),
		})
	}
	jsonOK(w, files)
}