package core

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/pwnduck/logger"
)

// OSType represents detected target OS
type OSType string

const (
	OSWindows OSType = "windows"
	OSMacOS   OSType = "macos"
	OSLinux   OSType = "linux"
	OSUnknown OSType = "unknown"
)

// DetectTargetOS detects the target OS via TTL fingerprinting
// Must be called after ethernet gadget is up and target has an IP
func DetectTargetOS(targetIP string) OSType {
	if targetIP == "" {
		targetIP = getUSBClientIP()
	}
	if targetIP == "" {
		logger.Warn(logger.SrcNetwork, "No target IP found for OS detection")
		return OSUnknown
	}

	ttl, err := getTTL(targetIP)
	if err != nil {
		logger.Warn(logger.SrcNetwork, "TTL detection failed: "+err.Error())
		return OSUnknown
	}

	os := ttlToOS(ttl)
	logger.Info(logger.SrcNetwork, fmt.Sprintf("OS detected: %s (TTL=%d IP=%s)", os, ttl, targetIP))
	return os
}

func ttlToOS(ttl int) OSType {
	switch {
	case ttl >= 120 && ttl <= 128:
		return OSWindows // Windows default TTL = 128
	case ttl >= 60 && ttl <= 64:
		// Mac and Linux both use TTL=64
		// Differentiate by checking for open port 22 (SSH on Mac by default)
		return OSLinux // Default to Linux, refine later
	case ttl >= 250:
		return OSUnknown // Some network devices
	default:
		return OSUnknown
	}
}

func getTTL(ip string) (int, error) {
	out, err := exec.Command("ping", "-c", "1", "-W", "1", ip).Output()
	if err != nil {
		return 0, err
	}

	// Parse TTL from ping output
	// "64 bytes from 192.168.x.x: icmp_seq=1 ttl=64 time=0.123 ms"
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "ttl=") || strings.Contains(line, "TTL=") {
			lower := strings.ToLower(line)
			idx := strings.Index(lower, "ttl=")
			if idx == -1 {
				continue
			}
			rest := lower[idx+4:]
			var ttl int
			fmt.Sscanf(rest, "%d", &ttl)
			if ttl > 0 {
				return ttl, nil
			}
		}
	}
	return 0, fmt.Errorf("TTL not found in ping output")
}

// getUSBClientIP finds the IP of the USB ethernet client
func getUSBClientIP() string {
	// Check ARP table for usb0 interface
	out, err := exec.Command("arp", "-n", "-i", "usb0").Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "usb0") && !strings.Contains(line, "incomplete") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				ip := fields[0]
				if net.ParseIP(ip) != nil {
					return ip
				}
			}
		}
	}
	return ""
}

// WaitForUSBClient waits for a USB ethernet client to connect (max timeout)
func WaitForUSBClient(timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ip := getUSBClientIP(); ip != "" {
			return ip, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", fmt.Errorf("no USB ethernet client connected within %s", timeout)
}
