package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pwnduck/api"
	"github.com/pwnduck/core"
	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

func main() {
	port := flag.String("port", "1337", "HTTP server port")
	flag.Parse()

	// Dev mode — if /opt/pwnduck doesn't exist, use ./pwnduck-data
	base := "/opt/pwnduck"
	if _, err := os.Stat(base); os.IsNotExist(err) {
		base = "./pwnduck-data"
		fmt.Println("[DEV] Running in dev mode — data dir:", base)
	}

	// Create all required directories
	dirs := []string{
		base + "/payload",
		base + "/loot",
		base + "/www",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			log.Fatal("mkdir:", err)
		}
	}

	// Init logger first — everything else logs
	logger.Init(base + "/logs.json")
	logger.Info(logger.SrcSystem, "PwnDuck starting...")

	// Init store
	if err := store.InitConfig(base + "/config.json"); err != nil {
		log.Fatal("config init:", err)
	}
	if err := store.InitPayloads(base + "/payload"); err != nil {
		log.Fatal("payload init:", err)
	}
	if err := store.InitLoot(base + "/loot"); err != nil {
		log.Fatal("loot init:", err)
	}

	// Migrate old payloads.json if it exists
	store.MigrateOld(base + "/payloads.json")

	// Setup USB gadget (only on real hardware)
	if core.GadgetAvailable() {
		fmt.Println("[*] Setting up USB gadget...")
		if err := core.SetupGadget(); err != nil {
			logger.Error(logger.SrcGadget, "Gadget setup failed: "+err.Error())
			fmt.Println("[!] Gadget setup failed:", err.Error())
		} else {
			fmt.Println("[*] Gadget setup OK")
		}
	} else {
		logger.Warn(logger.SrcGadget, "Gadget not available (dev mode)")
	}

	// Start WiFi AP (only on real hardware)
	if _, err := os.Stat("/etc/hostapd/hostapd.conf"); err == nil || core.WifiRunning() {
		if err := core.StartWifi(); err != nil {
			logger.Error(logger.SrcWifi, "WiFi start failed: "+err.Error())
		}
	} else {
		logger.Warn(logger.SrcWifi, "WiFi not available (dev mode)")
	}

	// Start USB monitor in background
	go core.MonitorUSB()

	// Init API
	api.SetLibraryDir("./library")
	handler := api.NewServer(base + "/www")

	logger.Success(logger.SrcSystem, fmt.Sprintf("PwnDuck ready on port %s", *port))
	fmt.Printf("[*] PwnDuck running → http://0.0.0.0:%s\n", *port)
	fmt.Printf("[*] Base dir        → %s\n", base)
	fmt.Printf("[*] HID available   → %v\n", core.HIDAvailable())
	fmt.Printf("[*] Gadget          → %v\n", core.GadgetAvailable())

	if err := http.ListenAndServe(":"+*port, handler); err != nil {
		log.Fatal(err)
	}
}