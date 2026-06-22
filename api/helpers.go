package api

import (
	"encoding/json"
	"net/http"
)

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Status response used across handlers
type StatusResponse struct {
	HID            bool   `json:"hid"`
	Wifi           bool   `json:"wifi"`
	UDCState       string `json:"udc_state"`
	TriggerEnabled bool   `json:"trigger_enabled"`
	TriggerCount   int    `json:"trigger_count"`
	PayloadCount   int    `json:"payload_count"`
	LootCount      int    `json:"loot_count"`
}
