#!/bin/sh
# Startet Chromium im Kiosk-Modus auf dem Display des Raspberry Pi.
# Wird aus ~/.config/labwc/autostart (Raspberry Pi OS Bookworm, Wayland) aufgerufen.

URL="${1:-http://localhost:8080/display}"

# Warten, bis der Dienst antwortet (z.B. direkt nach dem Booten)
until curl -sf http://localhost:8080/api/health >/dev/null; do sleep 2; done

# Je nach Raspberry-Pi-OS-Version heisst das Paket chromium oder chromium-browser
BROWSER=$(command -v chromium || command -v chromium-browser)

exec "$BROWSER" \
  --kiosk --noerrdialogs --disable-infobars --no-first-run \
  --disable-session-crashed-bubble --disable-features=Translate \
  --check-for-update-interval=31536000 \
  --password-store=basic \
  "$URL"
