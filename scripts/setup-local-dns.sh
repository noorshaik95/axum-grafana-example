#!/usr/bin/env bash
set -euo pipefail

# Setup local DNS for *.slate.local → 127.0.0.1
# Supports macOS (dnsmasq via Homebrew) and Linux (/etc/hosts)

DOMAIN="slate.local"
SUBDOMAINS=(
  "admin"
  "app"
  "teach"
  "api"
  "traefik"
  "demo-university"
)

print_header() {
  echo "=========================================="
  echo "  Slate LMS — Local DNS Setup"
  echo "=========================================="
  echo ""
}

setup_macos() {
  echo "[macOS] Setting up dnsmasq for *.${DOMAIN} → 127.0.0.1"
  echo ""

  # Install dnsmasq if not present
  if ! command -v dnsmasq &>/dev/null; then
    echo "  Installing dnsmasq via Homebrew..."
    brew install dnsmasq
  else
    echo "  dnsmasq already installed."
  fi

  # Configure dnsmasq
  DNSMASQ_CONF_DIR="/opt/homebrew/etc/dnsmasq.d"
  if [ ! -d "$DNSMASQ_CONF_DIR" ]; then
    DNSMASQ_CONF_DIR="/usr/local/etc/dnsmasq.d"
  fi
  mkdir -p "$DNSMASQ_CONF_DIR"

  CONF_FILE="${DNSMASQ_CONF_DIR}/slate.conf"
  echo "  Writing config to ${CONF_FILE}"
  echo "address=/${DOMAIN}/127.0.0.1" > "$CONF_FILE"

  # Restart dnsmasq
  echo "  Restarting dnsmasq..."
  sudo brew services restart dnsmasq 2>/dev/null || sudo killall -HUP dnsmasq 2>/dev/null || true

  # Create resolver entry so macOS uses dnsmasq for *.slate.local
  RESOLVER_DIR="/etc/resolver"
  echo "  Setting up /etc/resolver/${DOMAIN} (requires sudo)..."
  sudo mkdir -p "$RESOLVER_DIR"
  echo "nameserver 127.0.0.1" | sudo tee "${RESOLVER_DIR}/${DOMAIN}" > /dev/null

  echo ""
  echo "  Done! All *.${DOMAIN} addresses now resolve to 127.0.0.1"
  echo ""
  echo "  Verify with: dscacheutil -q host -a name app.${DOMAIN}"
  echo "  Or:          ping -c1 app.${DOMAIN}"
}

setup_linux() {
  echo "[Linux] Adding entries to /etc/hosts for ${DOMAIN} subdomains"
  echo ""

  HOSTS_FILE="/etc/hosts"
  MARKER_START="# === Slate LMS local dev ==="
  MARKER_END="# === End Slate LMS ==="

  # Remove old entries if present
  if grep -q "$MARKER_START" "$HOSTS_FILE" 2>/dev/null; then
    echo "  Removing old Slate entries..."
    sudo sed -i "/${MARKER_START}/,/${MARKER_END}/d" "$HOSTS_FILE"
  fi

  echo "  Adding entries to ${HOSTS_FILE} (requires sudo)..."
  {
    echo "$MARKER_START"
    for sub in "${SUBDOMAINS[@]}"; do
      echo "127.0.0.1  ${sub}.${DOMAIN}"
    done
    echo "$MARKER_END"
  } | sudo tee -a "$HOSTS_FILE" > /dev/null

  echo ""
  echo "  Done! The following entries were added to ${HOSTS_FILE}:"
  for sub in "${SUBDOMAINS[@]}"; do
    echo "    127.0.0.1  ${sub}.${DOMAIN}"
  done
  echo ""
  echo "  Note: New tenant subdomains must be added manually on Linux,"
  echo "  or switch to dnsmasq/systemd-resolved for wildcard support."
  echo ""
  echo "  Verify with: ping -c1 app.${DOMAIN}"
}

# --- Main ---
print_header

case "$(uname -s)" in
  Darwin)
    setup_macos
    ;;
  Linux)
    setup_linux
    ;;
  *)
    echo "Unsupported OS: $(uname -s)"
    echo "Please manually add *.${DOMAIN} → 127.0.0.1 to your DNS."
    exit 1
    ;;
esac

echo ""
echo "Next steps:"
echo "  1. Start Traefik:  docker compose up -d traefik"
echo "  2. Open dashboard: http://traefik.${DOMAIN}"
echo "  3. Test routing:   curl http://api.${DOMAIN}/health"
