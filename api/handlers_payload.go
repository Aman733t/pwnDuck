package api

import (
	"net/http"

	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

func handlePayloads(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, store.GetAllPayloads())
}

func handlePayloadSave(w http.ResponseWriter, r *http.Request) {
	var p store.Payload
	if err := decode(r, &p); err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	if err := store.SavePayload(&p); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	logger.Info(logger.SrcSystem, "Payload saved: "+p.Name)
	jsonOK(w, map[string]any{"ok": true, "id": p.ID})
}

func handlePayloadDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	decode(r, &req)
	p, _ := store.GetPayloadByID(req.ID)
	if err := store.DeletePayload(req.ID); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	if p != nil {
		logger.Warn(logger.SrcSystem, "Payload deleted: "+p.Name)
	}
	jsonOK(w, map[string]bool{"ok": true})
}

func handleCategories(w http.ResponseWriter, r *http.Request) {
	meta := store.GetMeta()
	jsonOK(w, store.GetPayloadCategories(meta.Categories))
}

func handleCategorySave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	decode(r, &req)
	if req.Name == "" {
		jsonErr(w, "name required", 400)
		return
	}
	store.AddCategory(req.Name)
	jsonOK(w, map[string]bool{"ok": true})
}

func handleCategoryDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	decode(r, &req)
	store.RemoveCategory(req.Name)
	jsonOK(w, map[string]bool{"ok": true})
}

func handleTags(w http.ResponseWriter, r *http.Request) {
	meta := store.GetMeta()
	jsonOK(w, store.GetPayloadTags(meta.Tags))
}
