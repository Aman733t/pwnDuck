package core

import (
	"fmt"
	"strings"
	"os/exec"
	"time"

	"github.com/pwnduck/logger"
)

// OSType represents the detected target OS
type OSType string

const (
	OSWindows OSType = "windows"
	OSMacOS   OSType = "macos"
	OSLinux   OSType = "linux"
	OSUnknown OSType = "unknown"
)

// DetectOS detects target OS via TTL fingerprinting over usb0
func DetectOS() OSType {
	ip := EthernetClientIP()
	if ip == "" {
		logger.Warn(logger.SrcNetwork, "No ethernet client connected for OS detection")
		return OSUnknown
	}
	return DetectOSByIP(ip)
}

// DetectOSByIP detects OS of a specific IP via TTL
func DetectOSByIP(ip string) OSType {
	ttl, err := getTTL(ip)
	if err != nil {
		logger.Warn(logger.SrcNetwork, fmt.Sprintf("TTL ping failed for %s: %s", ip, err.Error()))
		return OSUnknown
	}

	os := ttlToOS(ttl)
	logger.Info(logger.SrcNetwork, fmt.Sprintf("OS detected: %s (TTL=%d IP=%s)", os, ttl, ip))
	return os
}

func ttlToOS(ttl int) OSType {
	switch {
	case ttl >= 120 && ttl <= 128:
		return OSWindows // Windows default TTL = 128
	case ttl >= 60 && ttl <= 64:
		return OSLinux // Linux/Mac TTL = 64 — refine later
	default:
		return OSUnknown
	}
}

func getTTL(ip string) (int, error) {
	out, err := exec.Command("ping", "-c", "1", "-W", "1", ip).Output()
	if err != nil {
		return 0, err
	}

	// Parse "ttl=64" from ping output
	lower := strings.ToLower(string(out))
	idx := strings.Index(lower, "ttl=")
	if idx == -1 {
		return 0, fmt.Errorf("TTL not found in ping output")
	}
	var ttl int
	fmt.Sscanf(lower[idx+4:], "%d", &ttl)
	if ttl == 0 {
		return 0, fmt.Errorf("failed to parse TTL")
	}
	return ttl, nil
}

// WaitForEthernetClient waits for a USB ethernet client to connect
func WaitForEthernetClient(timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ip := EthernetClientIP(); ip != "" {
			return ip, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", fmt.Errorf("no ethernet client connected within %s", timeout)
}

// EthernetStatus returns current ethernet gadget status
func EthernetStatus() map[string]any {
	clientIP := EthernetClientIP()
	connected := clientIP != ""

	status := map[string]any{
		"connected": connected,
		"client_ip": clientIP,
		"pi_ip":     "192.168.7.1",
	}

	if connected {
		status["os"] = string(DetectOSByIP(clientIP))
	} else {
		status["os"] = string(OSUnknown)
	}

	return status
}