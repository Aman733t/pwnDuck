# PwnDuck

A wireless HID injection tool built on Raspberry Pi Zero W. Control it from your phone via a WiFi hotspot and web dashboard.

Built from scratch as a learning project — Go, USB gadgets, HID injection, WiFi AP.

---

## Hardware

- Raspberry Pi Zero W
- MicroSD card (8GB+)
- Micro USB cable (OTG port — the middle port, not the power port)

> Upgrading to: Radxa Zero (4GB RAM, no eMMC) for faster boot and more power.

---

## Project Structure

```
pwnduck/
├── cmd/pwnduck/main.go        # Entry point
├── core/
│   ├── gadget.go              # USB gadget setup (HID + ethernet + mass storage)
│   ├── hid.go                 # HID injection + DuckyScript parser
│   ├── trigger.go             # USB monitor + auto trigger execution
│   ├── wifi.go                # WiFi AP management
│   └── network.go             # Ethernet gadget + OS detection
├── store/
│   ├── config.go              # App config (wifi, gadget, triggers)
│   ├── payload.go             # Payload storage (individual JSON files)
│   └── loot.go                # Loot file management
├── api/
│   ├── server.go              # HTTP server + routing
│   ├── handlers_payload.go    # /api/payloads/*
│   ├── handlers_inject.go     # /api/inject
│   ├── handlers_wifi.go       # /api/wifi/*
│   ├── handlers_trigger.go    # /api/triggers/*
│   ├── handlers_loot.go       # /api/loot/*
│   ├── handlers_library.go    # /api/library/*
│   ├── handlers_gadget.go     # /api/gadget/*
│   └── handlers_log.go        # /api/logs/* + SSE stream
├── logger/logger.go           # Structured logging + SSE broadcast
├── library/                   # Pre-built payload stubs (write your own)
│   ├── general/
│   ├── recon/
│   ├── credentials/
│   ├── remote_access/
│   └── exfiltration/
├── extensions/                # Extend PwnDuck with shell scripts
├── setup.sh                   # Fresh Pi setup script
├── Makefile
└── go.mod
```

---

## Runtime Data (on Pi)

```
/opt/pwnduck/
├── pwnduck          # compiled binary
├── config.json      # app config
├── logs.json        # event logs
├── payload/         # user saved payloads
├── loot/            # received files
├── library/         # payload library
└── www/             # React UI build
```

---

## Fresh Install

**Requirements:** Go 1.21+, Mac or Linux for building.

### 1. Flash Pi OS Lite on SD card

Use Raspberry Pi Imager. Enable SSH in settings.

### 2. Boot Pi and connect to home WiFi

### 3. Run setup script

```bash
make setup PI=192.168.x.x
```

### 4. Deploy binary + UI + library

```bash
make deploy-full PI=192.168.x.x
```

### 5. Reboot

```bash
ssh pi@192.168.x.x "sudo reboot"
```

### 6. Connect

- WiFi: `PwnDuck` / `password123`
- Dashboard: `http://10.0.0.1:1337`

---

## Build Commands

```bash
# Build for Pi Zero W (ARMv6)
make build

# Build for Radxa Zero (ARM64)
make build-radxa

# Build for local Mac (dev/testing)
make build-local

# Deploy binary only
make deploy PI=192.168.x.x

# Deploy everything (binary + library + UI)
make deploy-full PI=192.168.x.x

# Run setup script on Pi
make setup PI=192.168.x.x
```

---

## DuckyScript Reference

Payloads use DuckyScript syntax:

```
REM This is a comment
DELAY 1000          # wait 1 second
STRING Hello World  # type text
ENTER               # press Enter
GUI r               # Windows key + r
CTRL ALT DELETE     # key combo
TAB
BACKSPACE
ESC
F1 ... F12
UP / DOWN / LEFT / RIGHT
```

---

## Payload Library

The `library/` folder contains payload stubs organized by category:

| Category | Description |
|---|---|
| `general/` | Basic test payloads |
| `recon/` | System information gathering |
| `credentials/` | Credential related (write your own) |
| `remote_access/` | Remote access (write your own) |
| `exfiltration/` | File exfiltration (write your own) |

> All payloads in `credentials/`, `remote_access/`, and `exfiltration/` are empty stubs. Write your own scripts for authorised testing on your own machines only.

---

## Extensions

Extensions are shell scripts that run when a trigger fires.

```
extensions/
└── your_extension/
    ├── extension.json   # manifest
    └── run.sh           # entrypoint
```

`extension.json`:
```json
{
  "id": "your_extension",
  "name": "Your Extension",
  "version": "1.0",
  "description": "What it does",
  "entrypoint": "run.sh",
  "trigger": "USB_CONNECTED",
  "enabled": false
}
```

---

## USB Ports

Pi Zero W has two micro USB ports:

```
[PWR] ← power only (left)
[USB] ← OTG port   (right/middle) ← use this one
```

Always plug the OTG port into the target computer.

---

## Troubleshooting

**HID not available:**
- Make sure Pi is plugged into computer via OTG port
- Check `ls /sys/class/udc/` — should show `20980000.usb`
- Check `/boot/firmware/config.txt` has `dtoverlay=dwc2,dr_mode=peripheral` under `[all]`
- Make sure `dwc_otg` is blacklisted: `cat /etc/modprobe.d/blacklist-dwc_otg.conf`

**WiFi hotspot not showing:**
- Check `rc.local` ran: `sudo systemctl status rc-local`
- Check hostapd: `pgrep hostapd`
- Check wlan0 IP: `ip addr show wlan0`

**Dashboard not loading:**
- Check pwnduck service: `sudo systemctl status pwnduck`
- Check logs: `sudo journalctl -u pwnduck -n 50`

---

## ⚠️ Disclaimer

This tool is for educational purposes and authorised testing on your own devices only. Do not use on systems you do not own or have explicit permission to test.