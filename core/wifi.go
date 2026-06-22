package core

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

const hostapdConf = "/etc/hostapd/hostapd.conf"

// StartWifi starts the WiFi AP using config from store
func StartWifi() error {
	cfg := store.GetWifi()
	return applyWifiConfig(cfg)
}

// ApplyWifi writes new config and reloads hostapd
func ApplyWifi(cfg store.WifiConfig) error {
	if err := store.SetWifi(cfg); err != nil {
		return err
	}
	return applyWifiConfig(cfg)
}

func applyWifiConfig(cfg store.WifiConfig) error {
	conf := buildHostapdConf(cfg)
	if err := os.WriteFile(hostapdConf, []byte(conf), 0644); err != nil {
		return fmt.Errorf("write hostapd.conf: %w", err)
	}

	// Try SIGHUP first (reload without dropping connections)
	if err := exec.Command("pkill", "-HUP", "-x", "hostapd").Run(); err != nil {
		// Not running — start fresh
		exec.Command("pkill", "-x", "hostapd").Run()
		if err := exec.Command("hostapd", "-B", hostapdConf).Start(); err != nil {
			return fmt.Errorf("start hostapd: %w", err)
		}
	}

	logger.Success(logger.SrcWifi, fmt.Sprintf("WiFi AP: SSID=%s CH=%d AUTH=%s",
		cfg.SSID, cfg.Channel, cfg.Auth))
	return nil
}

// RestartWifi restarts the hostapd process
func RestartWifi() error {
	if err := exec.Command("pkill", "-HUP", "-x", "hostapd").Run(); err != nil {
		exec.Command("pkill", "-x", "hostapd").Run()
		cfg := store.GetWifi()
		if err := exec.Command("hostapd", "-B", hostapdConf).Start(); err != nil {
			return fmt.Errorf("restart hostapd: %w", err)
		}
		_ = cfg
	}
	logger.Info(logger.SrcWifi, "WiFi restarted")
	return nil
}

// WifiRunning checks if hostapd is running
func WifiRunning() bool {
	return exec.Command("pgrep", "-x", "hostapd").Run() == nil
}

func buildHostapdConf(cfg store.WifiConfig) string {
	hidden := "0"
	if cfg.Hidden {
		hidden = "1"
	}

	base := fmt.Sprintf(`interface=wlan0
driver=nl80211
ssid=%s
hw_mode=g
channel=%s
wmm_enabled=0
macaddr_acl=0
ignore_broadcast_ssid=%s
`, cfg.SSID, strconv.Itoa(cfg.Channel), hidden)

	switch cfg.Auth {
	case "WPA2":
		base += fmt.Sprintf(`auth_algs=1
wpa=2
wpa_passphrase=%s
wpa_key_mgmt=WPA-PSK
rsn_pairwise=CCMP
`, cfg.Password)
	case "WPA3":
		base += fmt.Sprintf(`auth_algs=1
wpa=2
wpa_passphrase=%s
wpa_key_mgmt=SAE
rsn_pairwise=CCMP
ieee80211w=2
`, cfg.Password)
	default: // Open
		base += "auth_algs=1\n"
	}

	return base
}
