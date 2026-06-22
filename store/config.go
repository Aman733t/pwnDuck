package store

import (
	"encoding/json"
	"os"
	"sync"
)

// AppConfig holds all persistent configuration
type AppConfig struct {
	Wifi    WifiConfig    `json:"wifi"`
	Gadget  GadgetConfig  `json:"gadget"`
	Trigger TriggerConfig `json:"trigger"`
	Meta    MetaConfig    `json:"meta"`
}

type WifiConfig struct {
	SSID     string `json:"ssid"`
	Password string `json:"password"`
	Channel  int    `json:"channel"`
	Auth     string `json:"auth"`
	Hidden   bool   `json:"hidden"`
}

type GadgetConfig struct {
	HID         bool   `json:"hid"`
	Ethernet    bool   `json:"ethernet"`
	MassStorage bool   `json:"mass_storage"`
	VendorID    string `json:"vendor_id"`
	ProductID   string `json:"product_id"`
	Manufacturer string `json:"manufacturer"`
	Product     string `json:"product"`
}

type TriggerConfig struct {
	Enabled  bool      `json:"enabled"`
	Triggers []Trigger `json:"triggers"`
}

type Trigger struct {
	ID        string  `json:"id"`
	PayloadID string  `json:"payload_id"`
	Delay     float64 `json:"delay"`
	Repeat    int     `json:"repeat"`
	Interval  float64 `json:"interval"`
	Enabled   bool    `json:"enabled"`
}

type MetaConfig struct {
	Categories []string `json:"categories"`
	Tags       []string `json:"tags"`
}

var (
	configFile string
	configMu   sync.RWMutex
	config     AppConfig
)

func InitConfig(path string) error {
	configFile = path
	return loadConfig()
}

func defaultConfig() AppConfig {
	return AppConfig{
		Wifi: WifiConfig{
			SSID:     "PwnDuck",
			Password: "password123",
			Channel:  6,
			Auth:     "WPA2",
			Hidden:   false,
		},
		Gadget: GadgetConfig{
			HID:          true,
			Ethernet:     false,
			MassStorage:  false,
			VendorID:     "0x1038",
			ProductID:    "0x1397",
			Manufacturer: "SteelSeries",
			Product:      "SteelSeries USB",
		},
		Trigger: TriggerConfig{
			Enabled:  false,
			Triggers: []Trigger{},
		},
		Meta: MetaConfig{
			Categories: []string{},
			Tags:       []string{},
		},
	}
}

func loadConfig() error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		config = defaultConfig()
		return saveConfig()
	}
	configMu.Lock()
	defer configMu.Unlock()
	if err := json.Unmarshal(data, &config); err != nil {
		config = defaultConfig()
		return saveConfig()
	}
	// Ensure slices are not nil
	if config.Trigger.Triggers == nil {
		config.Trigger.Triggers = []Trigger{}
	}
	if config.Meta.Categories == nil {
		config.Meta.Categories = []string{}
	}
	if config.Meta.Tags == nil {
		config.Meta.Tags = []string{}
	}
	return nil
}

func saveConfig() error {
	data, _ := json.MarshalIndent(config, "", "  ")
	tmp := configFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, configFile)
}

func GetConfig() AppConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return config
}

func GetWifi() WifiConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return config.Wifi
}

func SetWifi(w WifiConfig) error {
	configMu.Lock()
	config.Wifi = w
	configMu.Unlock()
	return saveConfig()
}

func GetGadget() GadgetConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return config.Gadget
}

func SetGadget(g GadgetConfig) error {
	configMu.Lock()
	config.Gadget = g
	configMu.Unlock()
	return saveConfig()
}

func GetTriggerConfig() TriggerConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return config.Trigger
}

func SetTriggerConfig(t TriggerConfig) error {
	configMu.Lock()
	config.Trigger = t
	configMu.Unlock()
	return saveConfig()
}

func GetMeta() MetaConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return config.Meta
}

func AddCategory(name string) error {
	configMu.Lock()
	for _, c := range config.Meta.Categories {
		if c == name {
			configMu.Unlock()
			return nil
		}
	}
	config.Meta.Categories = append(config.Meta.Categories, name)
	configMu.Unlock()
	return saveConfig()
}

func RemoveCategory(name string) error {
	configMu.Lock()
	cats := make([]string, 0)
	for _, c := range config.Meta.Categories {
		if c != name {
			cats = append(cats, c)
		}
	}
	config.Meta.Categories = cats
	configMu.Unlock()
	return saveConfig()
}
