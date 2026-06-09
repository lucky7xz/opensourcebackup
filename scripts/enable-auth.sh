#!/usr/bin/env bash
#
# OpenSourceBackup — enable RBAC authentication on the control plane.
#
# Sets a NEW admin login (email + password you choose) in the systemd
# EnvironmentFile and restarts the service so RBAC is enforced. The password
# is read interactively and is never echoed, never written to shell history,
# and never logged.
#
# Run as root on the control-plane host:
#   bash enable-auth.sh
#
# Background: when ADMIN_EMAIL is unset the server runs with auth DISABLED
# (every request is treated as a synthetic admin). Setting ADMIN_EMAIL +
# ADMIN_PASSWORD enables login. The bootstrap creates the admin user only if
# that email does not already exist — so use a NEW email to be sure a fresh,
# known-password admin is created.

set -euo pipefail

ENVFILE=/etc/opensourcebackup/server.env
SERVICE=opensourcebackup
MIN_LEN=10

if [ "$(id -u)" -ne 0 ]; then
  echo "Bitte als root ausführen (die Env-Datei gehört root)." >&2
  exit 1
fi
if [ ! -f "$ENVFILE" ]; then
  echo "Env-Datei nicht gefunden: $ENVFILE" >&2
  exit 1
fi

read -rp "Admin-Email [toto27021984@gmail.com]: " EMAIL
EMAIL="${EMAIL:-toto27021984@gmail.com}"

read -rsp "Neues Admin-Passwort (min ${MIN_LEN} Zeichen): " PW1; echo
read -rsp "Passwort wiederholen: " PW2; echo
if [ "$PW1" != "$PW2" ]; then
  echo "Passwörter stimmen nicht überein — abgebrochen." >&2
  exit 1
fi
if [ "${#PW1}" -lt "$MIN_LEN" ]; then
  echo "Passwort zu kurz (min ${MIN_LEN}) — abgebrochen." >&2
  exit 1
fi
case "$PW1" in
  *'"'*) echo 'Bitte kein doppeltes Anführungszeichen (") im Passwort verwenden.' >&2; exit 1 ;;
esac

# Backup, then rewrite ADMIN_EMAIL / ADMIN_PASSWORD lines (others untouched).
cp "$ENVFILE" "${ENVFILE}.bak-$(date +%Y%m%d-%H%M%S)"
TMP="$(mktemp)"
grep -vE '^(ADMIN_EMAIL|ADMIN_PASSWORD)=' "$ENVFILE" > "$TMP" || true
printf 'ADMIN_EMAIL=%s\n'    "$EMAIL" >> "$TMP"
printf 'ADMIN_PASSWORD=%s\n' "$PW1"   >> "$TMP"
install -m 600 -o root -g root "$TMP" "$ENVFILE"
rm -f "$TMP"
unset PW1 PW2

echo "→ server.env aktualisiert (ADMIN_EMAIL=$EMAIL, Passwort gesetzt, 600 root)."

systemctl daemon-reload
systemctl restart "$SERVICE"
sleep 3

echo "Service: $(systemctl is-active "$SERVICE")"
CODE="$(curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/v1/systems)"
echo "/v1/systems ohne Login → HTTP ${CODE}  (erwartet: 401)"
if [ "$CODE" = "401" ]; then
  echo "✓ Auth ist aktiv. Login: ${EMAIL} + dein neues Passwort."
else
  echo "⚠ Erwartet 401, bekam ${CODE}. Logs prüfen: journalctl -u ${SERVICE} -n 30 --no-pager"
fi
