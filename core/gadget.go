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
)

// SetupGadget configures USB gadget based on GadgetConfig
func SetupGadget() error {
	cfg := store.GetGadget()

	// Load libcomposite
	if err := exec.Command("modprobe", "libcomposite").Run(); err != nil {
		logger.Warn(logger.SrcGadget, "modprobe libcomposite: "+err.Error())
	}

	// Cleanup any existing gadget
	if err := teardownGadget(); err != nil {
		logger.Warn(logger.SrcGadget, "teardown: "+err.Error())
	}

	// Wait for configfs
	time.Sleep(100 * time.Millisecond)

	// Create gadget base
	if err := os.MkdirAll(gadgetBase, 0755); err != nil {
		return fmt.Errorf("create gadget dir: %w", err)
	}

	// USB IDs
	writeFile(gadgetBase+"/idVendor", cfg.VendorID)
	writeFile(gadgetBase+"/idProduct", cfg.ProductID)
	writeFile(gadgetBase+"/bcdDevice", "0x0100")
	writeFile(gadgetBase+"/bcdUSB", "0x0200")

	// Strings
	os.MkdirAll(gadgetBase+"/strings/0x409", 0755)
	writeFile(gadgetBase+"/strings/0x409/serialnumber", "SS9837456312A")
	writeFile(gadgetBase+"/strings/0x409/manufacturer", cfg.Manufacturer)
	writeFile(gadgetBase+"/strings/0x409/product", cfg.Product)

	// Config
	os.MkdirAll(gadgetBase+"/configs/c.1/strings/0x409", 0755)
	writeFile(gadgetBase+"/configs/c.1/strings/0x409/configuration", "PwnDuck Config")
	writeFile(gadgetBase+"/configs/c.1/MaxPower", "250")

	// HID function
	if cfg.HID {
		if err := setupHIDFunction(); err != nil {
			return fmt.Errorf("setup HID: %w", err)
		}
	}

	// Ethernet function (RNDIS for Windows, ECM for Mac/Linux)
	if cfg.Ethernet {
		if err := setupEthernetFunction(); err != nil {
			logger.Warn(logger.SrcGadget, "setup ethernet: "+err.Error())
		}
	}

	// Mass Storage function
	if cfg.MassStorage {
		if err := setupMassStorageFunction(); err != nil {
			logger.Warn(logger.SrcGadget, "setup mass storage: "+err.Error())
		}
	}

	// Wait for kernel to register functions before binding
	time.Sleep(500 * time.Millisecond)

	// Bind to UDC
	udc, err := getUDC()
	if err != nil {
		return fmt.Errorf("get UDC: %w", err)
	}
	logger.Info(logger.SrcGadget, "Binding to UDC: "+udc)
	if err := writeFile(gadgetBase+"/UDC", udc); err != nil {
		return fmt.Errorf("bind UDC: %w", err)
	}

	logger.Success(logger.SrcGadget, fmt.Sprintf("Gadget ready (HID=%v ETH=%v UMS=%v UDC=%s)",
		cfg.HID, cfg.Ethernet, cfg.MassStorage, udc))
	return nil
}

func setupHIDFunction() error {
	hidDir := gadgetBase + "/functions/hid.usb0"
	os.MkdirAll(hidDir, 0755)
	writeFile(hidDir+"/protocol", "1")
	writeFile(hidDir+"/subclass", "1")
	writeFile(hidDir+"/report_length", "8")

	// HID report descriptor for keyboard
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
		return err
	}

	return os.Symlink(hidDir, gadgetBase+"/configs/c.1/hid.usb0")
}

func setupEthernetFunction() error {
	ecmDir := gadgetBase + "/functions/ecm.usb0"
	os.MkdirAll(ecmDir, 0755)
	// MAC addresses for the gadget
	writeFile(ecmDir+"/dev_addr", "42:61:64:55:53:42") // "BadUSB"
	writeFile(ecmDir+"/host_addr", "48:6f:73:74:50:43") // "HostPC"
	return os.Symlink(ecmDir, gadgetBase+"/configs/c.1/ecm.usb0")
}

func setupMassStorageFunction() error {
	umsDir := gadgetBase + "/functions/mass_storage.usb0"
	os.MkdirAll(umsDir, 0755)
	writeFile(umsDir+"/stall", "0")
	writeFile(umsDir+"/lun.0/removable", "1")
	// Image file for mass storage — create a small one if not exists
	imgPath := "/opt/pwnduck/ums.img"
	if _, err := os.Stat(imgPath); os.IsNotExist(err) {
		// Create 32MB image
		exec.Command("dd", "if=/dev/zero", "of="+imgPath, "bs=1M", "count=32").Run()
		exec.Command("mkfs.fat", imgPath, "-F", "32", "-I").Run()
	}
	writeFile(umsDir+"/lun.0/file", imgPath)
	return os.Symlink(umsDir, gadgetBase+"/configs/c.1/mass_storage.usb0")
}

func teardownGadget() error {
	if _, err := os.Stat(gadgetBase); os.IsNotExist(err) {
		return nil
	}

	// Unbind from UDC
	writeFile(gadgetBase+"/UDC", "")
	time.Sleep(100 * time.Millisecond)

	// Remove symlinks
	for _, link := range []string{"hid.usb0", "ecm.usb0", "mass_storage.usb0"} {
		os.Remove(gadgetBase + "/configs/c.1/" + link)
	}

	// Remove dirs in reverse order
	dirs := []string{
		gadgetBase + "/configs/c.1/strings/0x409",
		gadgetBase + "/configs/c.1",
		gadgetBase + "/functions/hid.usb0",
		gadgetBase + "/functions/ecm.usb0",
		gadgetBase + "/functions/mass_storage.usb0",
		gadgetBase + "/strings/0x409",
		gadgetBase,
	}
	for _, d := range dirs {
		os.Remove(d)
	}
	return nil
}

func getUDC() (string, error) {
	entries, err := os.ReadDir(udcPath)
	if err != nil {
		return "", fmt.Errorf("read UDC dir: %w", err)
	}
	for _, e := range entries {
		// UDC entries are symlinks, not regular dirs
		return e.Name(), nil
	}
	return "", fmt.Errorf("no UDC found in %s", udcPath)
}

func writeFile(path, value string) error {
	return os.WriteFile(path, []byte(strings.TrimSpace(value)+"\n"), 0644)
}

// GadgetAvailable checks if configfs is mounted
func GadgetAvailable() bool {
	_, err := os.Stat("/sys/kernel/config")
	return err == nil
}