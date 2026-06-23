package core

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pwnduck/logger"
	"github.com/pwnduck/store"
)

const (
	gadgetBase = "/sys/kernel/config/usb_gadget/pwnduck"
	udcPath    = "/sys/class/udc"
	umsImgPath = "/opt/pwnduck/ums.img"
	umsImgSize = "64"
	bridgeName = "usbeth" // bridge between RNDIS + ECM
)

func SetupGadget() error {
	cfg := store.GetGadget()

	// RNDIS must load before other functions for Windows compatibility
	// Load in correct order: rndis first, then others
	exec.Command("modprobe", "usb_f_rndis").Run()
	exec.Command("modprobe", "usb_f_ecm").Run()
	exec.Command("modprobe", "usb_f_hid").Run()
	exec.Command("modprobe", "usb_f_mass_storage").Run()
	exec.Command("modprobe", "libcomposite").Run()

	// Cleanup any existing gadget
	teardownGadget()
	time.Sleep(200 * time.Millisecond)

	// Create gadget base
	if err := os.MkdirAll(gadgetBase, 0755); err != nil {
		return fmt.Errorf("create gadget dir: %w", err)
	}

	// USB IDs
	writeFile(gadgetBase+"/idVendor", cfg.VendorID)
	writeFile(gadgetBase+"/idProduct", cfg.ProductID)
	writeFile(gadgetBase+"/bcdDevice", "0x0100")
	writeFile(gadgetBase+"/bcdUSB", "0x0200")

	// Composite device class — CRITICAL for Mac/Windows to enumerate all functions
	// Without these, host only loads driver for first recognized function
	writeFile(gadgetBase+"/bDeviceClass", "0xEF")
	writeFile(gadgetBase+"/bDeviceSubClass", "0x02")
	writeFile(gadgetBase+"/bDeviceProtocol", "0x01")

	// OS Descriptors — required for Windows to auto-install RNDIS driver
	if cfg.Ethernet {
		writeFile(gadgetBase+"/os_desc/use", "1")
		writeFile(gadgetBase+"/os_desc/b_vendor_code", "0xbc")
		writeFile(gadgetBase+"/os_desc/qw_sign", "MSFT100")
	}

	// Device strings
	os.MkdirAll(gadgetBase+"/strings/0x409", 0755)
	writeFile(gadgetBase+"/strings/0x409/serialnumber", "SS9837456312A")
	writeFile(gadgetBase+"/strings/0x409/manufacturer", cfg.Manufacturer)
	writeFile(gadgetBase+"/strings/0x409/product", cfg.Product)

	// Config
	os.MkdirAll(gadgetBase+"/configs/c.1/strings/0x409", 0755)
	writeFile(gadgetBase+"/configs/c.1/strings/0x409/configuration", "PwnDuck")
	writeFile(gadgetBase+"/configs/c.1/MaxPower", "250")
	writeFile(gadgetBase+"/configs/c.1/bmAttributes", "0x80")

	// OS descriptor must point to config
	if cfg.Ethernet {
		os.Symlink(gadgetBase+"/configs/c.1", gadgetBase+"/os_desc/c.1")
	}

	// ---- RNDIS must be FIRST for Windows ----
	if cfg.Ethernet {
		if err := setupRNDISFunction(); err != nil {
			logger.Warn(logger.SrcGadget, "RNDIS setup failed: "+err.Error())
		} else {
			logger.Info(logger.SrcGadget, "RNDIS function configured (Windows)")
		}
	}

	// ---- HID ----
	if cfg.HID {
		if err := setupHIDFunction(); err != nil {
			return fmt.Errorf("setup HID: %w", err)
		}
		logger.Info(logger.SrcGadget, "HID function configured")
	}

	// ---- ECM (Mac/Linux) ----
	if cfg.Ethernet {
		if err := setupECMFunction(); err != nil {
			logger.Warn(logger.SrcGadget, "ECM setup failed: "+err.Error())
		} else {
			logger.Info(logger.SrcGadget, "ECM function configured (Mac/Linux)")
		}
	}

	// ---- Mass Storage ----
	if cfg.MassStorage {
		if err := setupMassStorageFunction(); err != nil {
			logger.Warn(logger.SrcGadget, "Mass storage setup failed: "+err.Error())
		} else {
			logger.Info(logger.SrcGadget, "Mass Storage function configured")
		}
	}

	// Wait for kernel to register all functions
	time.Sleep(500 * time.Millisecond)

	// Bind to UDC
	udc, err := getUDC()
	if err != nil {
		return fmt.Errorf("get UDC: %w", err)
	}
	logger.Info(logger.SrcGadget, "Binding to UDC: "+udc)
	if err := os.WriteFile(gadgetBase+"/UDC", []byte(udc), 0644); err != nil {
		return fmt.Errorf("bind UDC %s: %w", udc, err)
	}

	// Wait for gadget to enumerate
	time.Sleep(300 * time.Millisecond)

	// Bring up ethernet bridge after gadget is ready
	if cfg.Ethernet {
		go bringUpEthernetBridge()
	}

	logger.Success(logger.SrcGadget, fmt.Sprintf(
		"Gadget ready — HID=%v ETH=%v UMS=%v UDC=%s",
		cfg.HID, cfg.Ethernet, cfg.MassStorage, udc,
	))
	return nil
}

// ---- HID ----

func setupHIDFunction() error {
	hidDir := gadgetBase + "/functions/hid.usb0"
	os.MkdirAll(hidDir, 0755)
	writeFile(hidDir+"/protocol", "1")
	writeFile(hidDir+"/subclass", "1")
	writeFile(hidDir+"/report_length", "8")

	descriptor := []byte{
		0x05, 0x01, 0x09, 0x06, 0xa1, 0x01, 0x05, 0x07,
		0x19, 0xe0, 0x29, 0xe7, 0x15, 0x00, 0x25, 0x01,
		0x75, 0x01, 0x95, 0x08, 0x81, 0x02, 0x95, 0x01,
		0x75, 0x08, 0x81, 0x03, 0x95, 0x05, 0x75, 0x01,
		0x05, 0x08, 0x19, 0x01, 0x29, 0x05, 0x91, 0x02,
		0x95, 0x01, 0x75, 0x03, 0x91, 0x03, 0x95, 0x06,
		0x75, 0x08, 0x15, 0x00, 0x25, 0x65, 0x05, 0x07,
		0x19, 0x00, 0x29, 0x65, 0x81, 0x00, 0xc0,
	}
	if err := os.WriteFile(hidDir+"/report_desc", descriptor, 0644); err != nil {
		return fmt.Errorf("write report_desc: %w", err)
	}
	return os.Symlink(hidDir, gadgetBase+"/configs/c.1/hid.usb0")
}

// ---- RNDIS (Windows) ----

func setupRNDISFunction() error {
	rndisDir := gadgetBase + "/functions/rndis.usb0"
	if err := os.MkdirAll(rndisDir, 0755); err != nil {
		return err
	}
	// MAC addresses for RNDIS
	writeFile(rndisDir+"/dev_addr", "42:63:65:56:34:12")
	writeFile(rndisDir+"/host_addr", "42:63:65:12:34:56")

	// Windows OS descriptor for RNDIS
	os.MkdirAll(rndisDir+"/os_desc/interface.rndis", 0755)
	writeFile(rndisDir+"/os_desc/interface.rndis/compatible_id", "RNDIS")
	writeFile(rndisDir+"/os_desc/interface.rndis/sub_compatible_id", "5162001")

	return os.Symlink(rndisDir, gadgetBase+"/configs/c.1/rndis.usb0")
}

// ---- ECM (Mac/Linux) ----

func setupECMFunction() error {
	ecmDir := gadgetBase + "/functions/ecm.usb0"
	if err := os.MkdirAll(ecmDir, 0755); err != nil {
		return err
	}
	writeFile(ecmDir+"/dev_addr", "42:63:66:56:34:12")
	writeFile(ecmDir+"/host_addr", "42:63:66:12:34:56")
	return os.Symlink(ecmDir, gadgetBase+"/configs/c.1/ecm.usb0")
}

// ---- Ethernet Bridge ----

// bringUpEthernetBridge creates a bridge between RNDIS + ECM
// so we only deal with one interface (usbeth) regardless of host OS
func bringUpEthernetBridge() {
	// Wait for interfaces to appear
	ifaces := []string{"rndis0", "usb0"} // rndis0=Windows, usb0=Mac/Linux
	for i := 0; i < 30; i++ {
		found := 0
		for _, iface := range ifaces {
			if _, err := os.Stat("/sys/class/net/" + iface); err == nil {
				found++
			}
		}
		if found > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Create bridge
	exec.Command("ip", "link", "add", bridgeName, "type", "bridge").Run()
	exec.Command("ip", "link", "set", bridgeName, "up").Run()
	exec.Command("ip", "addr", "add", "192.168.7.1/24", "dev", bridgeName).Run()

	// Add available interfaces to bridge
	for _, iface := range ifaces {
		if _, err := os.Stat("/sys/class/net/" + iface); err == nil {
			exec.Command("ip", "link", "set", iface, "up").Run()
			exec.Command("ip", "link", "set", iface, "master", bridgeName).Run()
			logger.Info(logger.SrcNetwork, iface+" added to bridge "+bridgeName)
		}
	}

	logger.Info(logger.SrcNetwork, bridgeName+" up — 192.168.7.1/24")

	// Start DHCP on bridge
	exec.Command("dnsmasq",
		"--interface="+bridgeName,
		"--dhcp-range=192.168.7.2,192.168.7.10,12h",
		"--bind-interfaces",
		"--port=0",
		"--no-resolv",
		"--no-hosts",
	).Start()

	logger.Success(logger.SrcNetwork, "DHCP started on "+bridgeName+" (192.168.7.2-10)")
}

// EthernetClientIP returns connected host IP via bridge
func EthernetClientIP() string {
	// Check bridge first, then individual interfaces
	for _, iface := range []string{bridgeName, "usb0", "rndis0"} {
		out, err := exec.Command("arp", "-n", "-i", iface).Output()
		if err != nil {
			continue
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, iface) && !strings.Contains(line, "incomplete") {
				fields := strings.Fields(line)
				if len(fields) > 0 && fields[0] != "Address" {
					return fields[0]
				}
			}
		}
	}
	return ""
}

// ---- Mass Storage ----

func setupMassStorageFunction() error {
	if err := ensureUMSImage(); err != nil {
		return fmt.Errorf("create UMS image: %w", err)
	}

	umsDir := gadgetBase + "/functions/mass_storage.usb0"
	if err := os.MkdirAll(umsDir, 0755); err != nil {
		return err
	}
	writeFile(umsDir+"/stall", "0")
	writeFile(umsDir+"/lun.0/removable", "1")
	writeFile(umsDir+"/lun.0/ro", "0")
	writeFile(umsDir+"/lun.0/cdrom", "0")
	writeFile(umsDir+"/lun.0/file", umsImgPath)

	return os.Symlink(umsDir, gadgetBase+"/configs/c.1/mass_storage.usb0")
}

func ensureUMSImage() error {
	if _, err := os.Stat(umsImgPath); err == nil {
		return nil
	}
	logger.Info(logger.SrcGadget, fmt.Sprintf("Creating %sMB USB mass storage image...", umsImgSize))

	if err := exec.Command("dd",
		"if=/dev/zero", "of="+umsImgPath,
		"bs=1M", "count="+umsImgSize,
	).Run(); err != nil {
		return fmt.Errorf("dd: %w", err)
	}

	// Try mkfs.fat then mkfs.vfat as fallback
	if err := exec.Command("mkfs.fat", "-F", "32", "-n", "PWNDUCK", umsImgPath).Run(); err != nil {
		if err2 := exec.Command("mkfs.vfat", "-F", "32", "-n", "PWNDUCK", umsImgPath).Run(); err2 != nil {
			return fmt.Errorf("mkfs.fat: %w", err)
		}
	}
	logger.Success(logger.SrcGadget, "UMS image created ("+umsImgSize+"MB FAT32 PWNDUCK)")
	return nil
}

func MountUMSImage(mountPoint string) error {
	os.MkdirAll(mountPoint, 0755)
	return exec.Command("mount", "-o", "loop", umsImgPath, mountPoint).Run()
}

func UnmountUMSImage(mountPoint string) error {
	return exec.Command("umount", mountPoint).Run()
}

func UMSImagePath() string { return umsImgPath }

// ---- Teardown ----

func teardownGadget() error {
	if _, err := os.Stat(gadgetBase); os.IsNotExist(err) {
		return nil
	}

	// Unbind UDC
	os.WriteFile(gadgetBase+"/UDC", []byte(""), 0644)
	time.Sleep(200 * time.Millisecond)

	// Remove symlinks
	for _, link := range []string{"rndis.usb0", "hid.usb0", "ecm.usb0", "mass_storage.usb0"} {
		os.Remove(gadgetBase + "/configs/c.1/" + link)
	}
	os.Remove(gadgetBase + "/os_desc/c.1")

	// Remove dirs deepest first
	dirs := []string{
		gadgetBase + "/configs/c.1/strings/0x409",
		gadgetBase + "/configs/c.1",
		gadgetBase + "/functions/rndis.usb0/os_desc/interface.rndis",
		gadgetBase + "/functions/rndis.usb0/os_desc",
		gadgetBase + "/functions/rndis.usb0",
		gadgetBase + "/functions/hid.usb0",
		gadgetBase + "/functions/ecm.usb0",
		gadgetBase + "/functions/mass_storage.usb0",
		gadgetBase + "/strings/0x409",
		gadgetBase + "/os_desc",
		gadgetBase,
	}
	for _, d := range dirs {
		os.Remove(d)
	}

	// Teardown bridge
	exec.Command("ip", "link", "del", bridgeName).Run()

	return nil
}

// ---- Helpers ----

func getUDC() (string, error) {
	entries, err := os.ReadDir(udcPath)
	if err != nil {
		return "", fmt.Errorf("read UDC dir: %w", err)
	}
	for _, e := range entries {
		return e.Name(), nil
	}
	return "", fmt.Errorf("no UDC found in %s", udcPath)
}

func writeFile(path, value string) error {
	return os.WriteFile(path, []byte(strings.TrimSpace(value)+"\n"), 0644)
}

func GadgetAvailable() bool {
	_, err := os.Stat("/sys/kernel/config")
	return err == nil
}