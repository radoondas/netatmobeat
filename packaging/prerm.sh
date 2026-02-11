#!/bin/sh
set -e

# For deb: $1 is "remove" or "upgrade"
# For rpm: $1 is the number of remaining instances (0 = remove, 1+ = upgrade)
case "${1:-}" in
  remove|0)
    # Full removal — stop and disable
    if command -v systemctl >/dev/null 2>&1; then
      if systemctl is-active --quiet netatmobeat 2>/dev/null; then
        systemctl stop netatmobeat
      fi
      if systemctl is-enabled --quiet netatmobeat 2>/dev/null; then
        systemctl disable netatmobeat
      fi
    fi
    ;;
  upgrade|1|*)
    # Upgrade — only stop, do not disable (postinst will restart)
    if command -v systemctl >/dev/null 2>&1; then
      if systemctl is-active --quiet netatmobeat 2>/dev/null; then
        systemctl stop netatmobeat
      fi
    fi
    ;;
esac