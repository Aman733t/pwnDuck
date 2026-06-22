#!/bin/bash
# PwnDuck Setup Script
# Run this on a fresh Raspberry Pi Zero W after flashing Pi OS Lite
# Usage: sudo bash setup.sh

set -e

PWNDUCK_DIR="/opt/pwnduck"
SSID="PwnDuck"
PASSWORD="password123"

echo "================================================"
echo "  PwnDuck Setup"
echo "================================================"

# Must run as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root: sudo bash setup.sh"
  exit 1
fi

# ---- 1. System update ----
echo "[1/9] Updating system..."
apt update -q
apt install -y hostapd dnsmasq iw

# ---- 2. Disable conflicting services ----
echo "[2/9] Configuring services..."
systemctl stop hostapd 2>/dev/null || true
systemctl stop dnsmasq 2>/dev/null || true
systemctl disable hostapd 2>/dev/null || true
systemctl disable dnsmasq 2>/dev/null || true

# Prevent NetworkManager from managing wlan0
mkdir -p /etc/NetworkManager/conf.d
cat > /etc/NetworkManager/conf.d/unmanaged.conf << 'NMEOF'
[keyfile]
unmanaged-devices=interface-name:wlan0
NMEOF

# ---- 3. USB OTG setup ----
echo "[3/9] Configuring USB OTG..."

# Blacklist dwc_otg — it conflicts with dwc2 on Pi Zero W
echo "blacklist dwc_otg" > /etc/modprobe.d/blacklist-dwc_otg.conf

# Load dwc2 and libcomposite on boot
grep -qxF 'dwc2' /etc/modules || echo 'dwc2' >> /etc/modules
grep -qxF 'libcomposite' /etc/modules || echo 'libcomposite' >> /etc/modules

# Fix config.txt — add dwc2 overlay under [all] section only
CONFIG="/boot/firmware/config.txt"

# Remove any existing dwc2 lines to avoid duplicates
sed -i '/dtoverlay=dwc2/d' "$CONFIG"

# Add dwc2 peripheral mode under [all] section
if grep -q '^\[all\]' "$CONFIG"; then
  # Insert after [all]
  sed -i '/^\[all\]/a dtoverlay=dwc2,dr_mode=peripheral' "$CONFIG"
else
  # Append [all] section at end
  echo '' >> "$CONFIG"
  echo '[all]' >> "$CONFIG"
  echo 'dtoverlay=dwc2,dr_mode=peripheral' >> "$CONFIG"
fi

echo "config.txt updated:"
grep -A2 '\[all\]' "$CONFIG"

# ---- 4. Create directory structure ----
echo "[4/9] Creating directories..."
mkdir -p $PWNDUCK_DIR/payload
mkdir -p $PWNDUCK_DIR/loot
mkdir -p $PWNDUCK_DIR/www
mkdir -p $PWNDUCK_DIR/library/general/hello_world
mkdir -p $PWNDUCK_DIR/library/recon/windows_sysinfo
mkdir -p $PWNDUCK_DIR/library/credentials/windows_browser_creds
mkdir -p $PWNDUCK_DIR/library/credentials/windows_wifi_pass
mkdir -p $PWNDUCK_DIR/library/remote_access/windows_reverse_shell
mkdir -p $PWNDUCK_DIR/library/remote_access/macos_reverse_shell
mkdir -p $PWNDUCK_DIR/library/exfiltration/windows_docs
mkdir -p $PWNDUCK_DIR/library/exfiltration/windows_wifi_passwords
mkdir -p $PWNDUCK_DIR/library/exfiltration/macos_ssh_keys
mkdir -p $PWNDUCK_DIR/library/exfiltration/linux_passwd

# ---- 5. Write default config ----
echo "[5/9] Writing config..."
cat > $PWNDUCK_DIR/config.json << 'CFGEOF'
{
  "wifi": {
    "ssid": "PwnDuck",
    "password": "password123",
    "channel": 6,
    "auth": "WPA2",
    "hidden": false
  },
  "gadget": {
    "hid": true,
    "ethernet": false,
    "mass_storage": false,
    "vendor_id": "0x1038",
    "product_id": "0x1397",
    "manufacturer": "SteelSeries",
    "product": "SteelSeries USB"
  },
  "trigger": {
    "enabled": false,
    "triggers": []
  },
  "meta": {
    "categories": [],
    "tags": []
  }
}
CFGEOF

# ---- 6. hostapd config ----
echo "[6/9] Configuring WiFi AP..."
cat > /etc/hostapd/hostapd.conf << 'HAEOF'
interface=wlan0
driver=nl80211
ssid=PwnDuck
hw_mode=g
channel=6
wmm_enabled=0
macaddr_acl=0
ignore_broadcast_ssid=0
auth_algs=1
wpa=2
wpa_passphrase=password123
wpa_key_mgmt=WPA-PSK
rsn_pairwise=CCMP
HAEOF

# ---- 7. rc.local for WiFi AP on boot ----
echo "[7/9] Setting up WiFi AP autostart..."
cat > /etc/rc.local << 'RCEOF'
#!/bin/bash
sleep 10
iw reg set IN
ip link set wlan0 up
ip addr add 10.0.0.1/24 dev wlan0 2>/dev/null || true
hostapd -B /etc/hostapd/hostapd.conf
dnsmasq --interface=wlan0 --dhcp-range=10.0.0.2,10.0.0.20,12h --bind-interfaces
exit 0
RCEOF
chmod +x /etc/rc.local
systemctl enable rc-local 2>/dev/null || true

# ---- 8. systemd service ----
echo "[8/9] Creating systemd service..."
cat > /etc/systemd/system/pwnduck.service << 'SVCEOF'
[Unit]
Description=PwnDuck
After=rc-local.service
Requires=rc-local.service

[Service]
Type=simple
ExecStart=/opt/pwnduck/pwnduck
WorkingDirectory=/opt/pwnduck
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable pwnduck

# ---- 9. Permissions ----
echo "[9/9] Setting permissions..."
chmod +x $PWNDUCK_DIR/pwnduck 2>/dev/null || true

# ---- Done ----
echo ""
echo "================================================"
echo "  Setup complete!"
echo "================================================"
echo ""
echo "Next steps (from your Mac):"
echo "  1. make build"
echo "  2. scp pwnduck pi@<ip>:/opt/pwnduck/pwnduck"
echo "  3. scp -r library/* pi@<ip>:/opt/pwnduck/library/"
echo "  4. scp -r pwnduck-ui/dist/* pi@<ip>:/opt/pwnduck/www/"
echo "  5. sudo reboot"
echo ""
echo "After reboot:"
echo "  WiFi SSID : PwnDuck"
echo "  Password  : password123"
echo "  Dashboard : http://10.0.0.1:1337"
echo ""