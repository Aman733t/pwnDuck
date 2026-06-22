package core

import (
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

// Event types
const (
	EventUSBConnected    = "USB_CONNECTED"
	EventUSBDisconnected = "USB_DISCONNECTED"
	EventServiceStart    = "SERVICE_START"
)

const udcStatePath = "/sys/class/udc/20980000.usb/state"

// ReadUDCState returns the current UDC state
func ReadUDCState() string {
	data, err := os.ReadFile(udcStatePath)
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

// MonitorUSB watches the UDC state and fires triggers on connect/disconnect
func MonitorUSB() {
	lastState := ReadUDCState()
	logger.Info(logger.SrcSystem, "USB monitor started — state: "+lastState)

	// Fire on boot if already connected
	if lastState == "configured" {
		logger.Info(logger.SrcSystem, "USB already connected on boot")
		time.Sleep(3 * time.Second)
		FireEvent(EventUSBConnected)
	}

	for {
		time.Sleep(time.Second)
		state := ReadUDCState()
		if state == lastState {
			continue
		}
		prev := lastState
		lastState = state

		switch {
		case state == "configured":
			logger.Info(logger.SrcSystem, "USB connected to host")
			FireEvent(EventUSBConnected)
		case prev == "configured":
			logger.Warn(logger.SrcSystem, "USB disconnected from host")
			FireEvent(EventUSBDisconnected)
		}
	}
}

// FireEvent fires all triggers registered for an event
func FireEvent(event string) {
	switch event {
	case EventUSBConnected:
		cfg := store.GetTriggerConfig()
		if !cfg.Enabled || len(cfg.Triggers) == 0 {
			return
		}
		logger.Info(logger.SrcTrigger, "Firing triggers for: "+event)
		go executeTriggers(cfg.Triggers)
	}
}

func executeTriggers(triggers []store.Trigger) {
	// Sort by delay so triggers fire in order
	sorted := make([]store.Trigger, len(triggers))
	copy(sorted, triggers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Delay < sorted[j].Delay
	})

	for _, t := range sorted {
		if !t.Enabled {
			continue
		}

		p, ok := store.GetPayloadByID(t.PayloadID)
		if !ok {
			logger.Error(logger.SrcTrigger, "Payload not found: "+t.PayloadID)
			continue
		}

		// Wait for delay
		if t.Delay > 0 {
			time.Sleep(time.Duration(t.Delay * float64(time.Second)))
		}

		repeat := t.Repeat
		if repeat < 1 {
			repeat = 1
		}

		logger.Info(logger.SrcTrigger, "Firing: "+p.Name)

		for i := 0; i < repeat; i++ {
			if err := RunDucky(p.Script); err != nil {
				logger.Error(logger.SrcTrigger, "Failed: "+err.Error())
			} else {
				logger.Success(logger.SrcTrigger, "Executed: "+p.Name)
			}
			if i < repeat-1 && t.Interval > 0 {
				time.Sleep(time.Duration(t.Interval * float64(time.Second)))
			}
		}
	}
}

// TestTriggers manually fires all triggers (for UI test button)
func TestTriggers() error {
	cfg := store.GetTriggerConfig()
	if len(cfg.Triggers) == 0 {
		return nil
	}
	go executeTriggers(cfg.Triggers)
	logger.Info(logger.SrcTrigger, "Manual trigger test fired")
	return nil
}
