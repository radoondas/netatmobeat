#!/bin/sh
set -e

# Create netatmobeat system user if it doesn't exist
if ! getent passwd netatmobeat >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin netatmobeat
fi

# Ensure config directory ownership
chown root:netatmobeat /etc/netatmobeat
chmod 750 /etc/netatmobeat
chown root:netatmobeat /etc/netatmobeat/netatmobeat.yml
chmod 640 /etc/netatmobeat/netatmobeat.yml

# Reload systemd and restart on upgrade
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
  # For deb: $1 is "configure" (both fresh install and upgrade)
  # For rpm: $1 is the number of instances (1 = fresh install, 2+ = upgrade)
  case "${1:-}" in
    2|3|4|5|6|7|8|9)
      # rpm upgrade — restart the service that prerm stopped
      systemctl restart netatmobeat || true
      ;;
    configure)
      # deb install/upgrade — restart if service was previously enabled
      if systemctl is-enabled --quiet netatmobeat 2>/dev/null; then
        systemctl restart netatmobeat || true
      fi
      ;;
  esac
fi