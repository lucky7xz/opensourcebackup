# OpenSourceBackup — Troubleshooting & Known Issues

> Dokumentiert bekannte Probleme, ihre Ursachen und Lösungswege.  
> Stand: 2026-06-01

---

## Inhaltsverzeichnis

1. [PostgreSQL: Disk Full / Recovery Mode](#1-postgresql-disk-full--recovery-mode)
2. [Dashboard: ERR_TOO_MANY_REDIRECTS](#2-dashboard-err_too_many_redirects)
3. [Dashboard: Leere weiße Seite](#3-dashboard-leere-weisse-seite)
4. [API-Calls gehen auf localhost:8080](#4-api-calls-gehen-auf-localhost8080)
5. [CSRF: POST/PUT/DELETE schlägt mit 403 fehl](#5-csrf-postputdelete-schlägt-mit-403-fehl)
6. [RBAC: Alle Daten verschwinden nach Login](#6-rbac-alle-daten-verschwinden-nach-login)
7. [Agent: Token revoked or invalid](#7-agent-token-revoked-or-invalid)
8. [Agent: Backup-Pfad Z:\ nicht erreichbar (Windows Service)](#8-agent-backup-pfad-z-nicht-erreichbar-windows-service)
9. [OPNsense: export Command not found](#9-opnsense-export-command-not-found)
10. [Restic: executable file not found in \$PATH](#10-restic-executable-file-not-found-in-path)
11. [CI: Go-Version zu alt](#11-ci-go-version-zu-alt)
12. [Migration: no change / Spalte fehlt](#12-migration-no-change--spalte-fehlt)
13. [Agent Windows: sc.exe create haengt sich auf](#13-agent-windows-scexe-create-haengt-sich-auf)
13. [Proxmox: Disk voll (100%)](#13-proxmox-disk-voll-100)
14. [Proxmox: Passwort vergessen / kein Login](#14-proxmox-passwort-vergessen--kein-login)
15. [Windows Agent als Dienst: Service nicht registriert](#15-windows-agent-als-dienst-service-nicht-registriert)
16. [TypeScript Build-Fehler: unused variables](#16-typescript-build-fehler-unused-variables)
17. [Vite base-Pfad: Assets laden nicht unter /ui/](#17-vite-base-pfad-assets-laden-nicht-unter-ui)

---

## 1. PostgreSQL: Disk Full / Recovery Mode

**Symptom:**
```
FATAL: the database system is in recovery mode (SQLSTATE 57P03)
PANIC: could not write to file "pg_logical/replorigin_checkpoint.tmp": No space left on device
```

**Ursache:**  
Die Proxmox-Root-Partition (`/dev/mapper/pve-root`) war zu 100% voll. PostgreSQL konnte keine WAL-Checkpoint-Dateien schreiben, stürzte ab und kam beim Neustart nicht mehr aus dem Recovery-Modus heraus.

**Lösung:**
```bash
# 1. Speicher freigeben
apt-get clean && apt-get autoremove -y
journalctl --vacuum-size=100M
rm -rf /tmp/osb-build

# 2. Alte Proxmox-Kernel entfernen (automatisch via autoremove)

# 3. Log-Größe dauerhaft begrenzen
echo -e "[Journal]\nSystemMaxUse=200M" >> /etc/systemd/journald.conf
systemctl restart systemd-journald

# 4. PostgreSQL-Container neu starten
docker restart opensourcebackup-postgres-1

# 5. Warten bis bereit
until docker exec opensourcebackup-postgres-1 pg_isready -U opensourcebackup; do sleep 2; done

# 6. Server neu starten
systemctl restart opensourcebackup
```

**Prävention:**  
- `df -h /` regelmäßig überwachen
- Log-Rotation konfigurieren (siehe Schritt 3)
- Alte Kernel mit `apt-get autoremove` entfernen

---

## 2. Dashboard: ERR_TOO_MANY_REDIRECTS

**Symptom:**  
Browser zeigt `ERR_TOO_MANY_REDIRECTS` wenn `/ui/` aufgerufen wird.

**Ursache:**  
Die RBAC-Middleware hat unauthentifizierte Requests auf `/ui/` weitergeleitet — aber `/ui/` selbst war nicht als public markiert, was eine Endlosschleife erzeugte.

**Lösung:**  
In `internal/api/rbac.go` `/ui/` explizit als immer öffentlich markieren:

```go
if isPublicPath(p) || strings.HasPrefix(p, "/ui/") || p == "/" {
    next.ServeHTTP(w, r)
    return
}
```

---

## 3. Dashboard: Leere weiße Seite

**Symptom:**  
Die Seite lädt, ist aber komplett weiß. `<head></head>` ist leer.

**Ursache:**  
Vite baut Assets unter `/assets/...` (absoluter Pfad), aber der Server serviert die UI unter `/ui/`. Der Browser konnte `/assets/index.js` nicht finden, da dieser Pfad nicht durch den `/ui/`-Handler lief.

**Lösung:**  
In `web/vite.config.ts` den base-Pfad setzen:

```ts
export default defineConfig({
  plugins: [react()],
  base: '/ui/',
})
```

Dadurch werden alle Asset-Referenzen als `/ui/assets/...` generiert.

---

## 4. API-Calls gehen auf localhost:8080

**Symptom:**  
Browser-Console zeigt:
```
Fetch API cannot load http://localhost:8080/v1/systems. 
Refused to connect because it violates the document's Content Security Policy.
```

**Ursache:**  
In `api.ts` war der Fallback-Wert `http://localhost:8080`:
```ts
const BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
```
Wenn `VITE_API_URL` nicht gesetzt war, wurden alle API-Calls auf localhost:8080 geleitet, was die CSP verletzte.

**Lösung:**
```ts
// Leer = relative URLs (gleicher Host + Port) — funktioniert wenn UI vom Control Plane serviert wird
const BASE = import.meta.env.VITE_API_URL || ''
```

---

## 5. CSRF: POST/PUT/DELETE schlägt mit 403 fehl

**Symptom:**  
Dashboard-Aktionen wie "Enrollment Token generieren" oder "Repository erstellen" schlagen fehl mit `403 Forbidden`. Backend-Log: `CSRF token mismatch`.

**Ursache:**  
Die Double-Submit-Cookie CSRF-Protection erwartet den `X-CSRF-Token` Header bei mutierenden Requests. Das Frontend schickte ihn nicht.

**Lösung:**  
In `api.ts` CSRF-Token aus dem Cookie lesen und bei POST/PUT/DELETE mitsenden:

```ts
function csrfToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)osb_csrf=([^;]+)/)
  return match ? decodeURIComponent(match[1]) : ''
}

function mutatingHeaders(): Record<string, string> {
  return { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken() }
}
```

---

## 6. RBAC: Alle Daten verschwinden nach Login

**Symptom:**  
Nach dem Aktivieren von `ADMIN_PASSWORD` in der `server.env` zeigt das Dashboard keine Systeme, Jobs oder Agents mehr an.

**Ursache:**  
Die RBAC-Middleware blockiert alle `/v1/`-Requests ohne gültige Session. `authEnabled` war auf `true` gesetzt sobald `ADMIN_PASSWORD` gesetzt war — auch ohne `ADMIN_EMAIL` und ohne Login-Seite.

**Lösung:**  
Auth-Enforcement nur wenn **beide** Variablen gesetzt sind:

```go
// Auth nur erzwingen wenn ADMIN_EMAIL + ADMIN_PASSWORD konfiguriert sind.
// Nur ADMIN_PASSWORD = legacy Dev-Mode, kein API-Blocking.
authEnabled := adminEmail != "" && adminPass != ""
```

---

## 7. Agent: Token revoked or invalid

**Symptom:**
```
"enrollment failed: enroll: agent: unauthorized — token revoked or invalid"
```

**Ursachen:**
1. Enrollment-Token ist abgelaufen (TTL: 30 Minuten)
2. Enrollment-Token wurde bereits einmalig verwendet
3. Agent-Token aus Datei ist nach DB-Crash nicht mehr in der DB vorhanden

**Lösung:**
```powershell
# 1. Alten Token löschen
Remove-Item "C:\ProgramData\opensourcebackup\agent-token" -Force -ErrorAction SilentlyContinue

# 2. Neuen Enrollment-Token im Dashboard holen:
#    Agents → + Enroll Agent → System auswählen → Token kopieren

# 3. Token setzen und Agent neu starten
$env:ENROLLMENT_TOKEN = "neuer-token"
& "C:\ProgramData\opensourcebackup\opensourcebackup-agent.exe" run
```

---

## 8. Agent: Backup-Pfad Z:\ nicht erreichbar (Windows Service)

**Symptom:**
```
restic init: exit status 1 — Fatal: create repository at Z:\OpenSourceBackup failed:
unable to open repository: mkdir \\?\Z:\OpenSourceBackup: Das System kann den angegebenen Pfad nicht finden.
```

**Ursache:**  
Windows Services laufen unter dem SYSTEM-Konto. Gemappte Netzlaufwerke (Z:, Y: etc.) sind benutzerspezifisch und für SYSTEM nicht verfügbar.

**Lösung:**  
UNC-Pfad statt Laufwerksbuchstabe verwenden:

```powershell
# Z: auflösen
net use Z:
# → \\192.168.1.50\Public

# Repository mit UNC-Pfad konfigurieren
[Environment]::SetEnvironmentVariable("RESTIC_REPO", "\\192.168.1.50\Public\OpenSourceBackup", "Machine")
$env:RESTIC_REPO = "\\192.168.1.50\Public\OpenSourceBackup"

# Dienst neu installieren (damit neuer Pfad eingebettet wird)
& "C:\ProgramData\opensourcebackup\opensourcebackup-agent.exe" uninstall
& "C:\ProgramData\opensourcebackup\opensourcebackup-agent.exe" install
& "C:\ProgramData\opensourcebackup\opensourcebackup-agent.exe" start
```

---

## 9. OPNsense: export Command not found

**Symptom:**
```
root@OPNsense:~ # export CONTROL_PLANE_URL="http://..."
export: Command not found.
```

**Ursache:**  
OPNsense verwendet `tcsh` als Standard-Shell. In `tcsh` gibt es kein `export`, sondern `setenv`.

**Lösung:**
```csh
# tcsh: setenv statt export
setenv CONTROL_PLANE_URL "http://192.168.0.100:8090"
setenv ENROLLMENT_TOKEN "token"
setenv RESTIC_PASSWORD "passwort"
setenv RESTIC_REPO "/var/backups/opensense"
setenv AGENT_TOKEN_FILE "/var/db/opensourcebackup/agent-token"
```

Für Heredocs in tcsh `/bin/sh` verwenden:
```csh
/bin/sh
# jetzt bash-Syntax verwenden
cat > /datei << 'EOF'
...
EOF
exit
```

---

## 10. Restic: executable file not found in $PATH

**Symptom:**
```
"error":"restic init: exec: \"restic\": executable file not found in $PATH"
```

**Ursache:**  
Restic ist nicht installiert oder nicht im `$PATH` des Agent-Prozesses.

**Lösung Windows:**
```powershell
$r = "0.17.3"
Invoke-WebRequest "https://github.com/restic/restic/releases/download/v$r/restic_${r}_windows_amd64.zip" `
  -OutFile "$env:TEMP\restic.zip"
Expand-Archive "$env:TEMP\restic.zip" -DestinationPath "C:\ProgramData\opensourcebackup" -Force
Get-ChildItem "C:\ProgramData\opensourcebackup\restic_*.exe" | Rename-Item -NewName "restic.exe"
[Environment]::SetEnvironmentVariable("RESTIC_BIN", "C:\ProgramData\opensourcebackup\restic.exe", "Machine")
```

**Lösung FreeBSD/OPNsense:**
```sh
fetch -o /usr/local/bin/restic.bz2 \
  https://github.com/restic/restic/releases/download/v0.17.3/restic_0.17.3_freebsd_amd64.bz2
bunzip2 /usr/local/bin/restic.bz2
mv /usr/local/bin/restic.bz2.out /usr/local/bin/restic 2>/dev/null || true
chmod 755 /usr/local/bin/restic
```

---

## 11. CI: Go-Version zu alt

**Symptom:**
```
go: golang.org/x/crypto@v0.52.0 requires go >= 1.25.0 (running go 1.22.0)
```

**Ursache:**  
Die GitHub Actions CI nutzte Go 1.22, aber `golang.org/x/crypto` v0.52.0 setzt Go 1.25+ voraus.

**Lösung:**  
In `.github/workflows/ci.yml` alle drei Jobs auf `go-version: "1.25"` setzen:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: "1.25"
    cache: true
```

Zusätzlich `go.mod` auf `go 1.25.0` aktualisieren.

---

## 12. Migration: no change / Spalte fehlt

**Symptom:**  
`migrate ... up` gibt `no change` aus, aber der Server crasht beim Start weil neue Spalten fehlen.

**Ursache:**  
Das `migrate`-Kommando wurde mit einer falsch extrahierten `DATABASE_URL` ausgeführt (leerer String). `no change` bedeutete: keine Migration lief.

**Lösung:**
```bash
# Explizit mit vollständiger URL aufrufen:
DB_PASS=$(cat /etc/opensourcebackup/.db_password)
migrate -path /tmp/osb-build/migrations \
  -database "postgres://opensourcebackup:${DB_PASS}@127.0.0.1:5432/opensourcebackup?sslmode=disable" \
  up

# Aktuellen Stand prüfen:
migrate ... version
# Erwartete Ausgabe: 18 (aktuelle Migrations-Anzahl)
```

---

## 13. Proxmox: Disk voll (100%)

**Symptom:**  
Alle Dienste hängen, Docker kann keine Container starten, SSH-Verbindungen sind langsam.

```bash
df -h /
# Filesystem  Size  Used  Avail  Use%
# /dev/mapper/pve-root  68G  68G  0  100%
```

**Hauptverursacher in diesem Projekt:**
- Systemd-Journal: ~1.5 GB unkomprimierte Logs
- Alte Proxmox-Kernel: ~1.1 GB pro Version
- `/tmp/osb-build`: ~160 MB pro Build-Klon

**Lösung:**
```bash
# Sofort-Freigabe:
apt-get clean && apt-get autoremove -y   # ~1.1 GB (alte Kernel)
journalctl --vacuum-size=100M            # ~1.2 GB (alte Logs)
rm -rf /tmp/osb-build                    # ~160 MB

# Dauerhaft:
echo -e "[Journal]\nSystemMaxUse=200M" >> /etc/systemd/journald.conf
systemctl restart systemd-journald
```

---

## 14. Proxmox: Passwort vergessen / kein Login

**Symptom:**  
Proxmox Web-UI zeigt "failed login". SSH mit Passwort schlägt fehl.

**Lösung 1 — SSH mit Passwort:** (wenn PAM und PVE-Auth divergieren)
```bash
# PVE-Passwort über SSH zurücksetzen (wenn SSH noch geht via Keys)
pveum passwd root@pam
# Neues Passwort eingeben, dann im Web mit Realm "Linux PAM" einloggen
```

**Lösung 2 — Falsches Realm:**  
Im Proxmox-Login-Dialog **Realm: "Linux PAM standard authentication"** auswählen, nicht "Proxmox VE authentication server".

**Lösung 3 — Passwort über Konsole zurücksetzen:**
```bash
# Direkt am Server oder über SSH
passwd root
# Danach im Web neu einloggen
```

---

## 15. Windows Agent als Dienst: Service nicht registriert

**Symptom:**
```
Get-Service : No service with name "OpenSourceBackupAgent" found.
```

Obwohl `install` ausgeführt wurde, erscheint der Dienst nicht in `services.msc`.

**Ursache:**  
`kardianos/service` schlägt bei der Windows-Service-Registrierung still fehl wenn `EnvVars` in der Konfiguration vorhanden sind und ein bestimmtes Windows-Setup vorliegt.

**Workaround — Task Scheduler:**
```powershell
$action = New-ScheduledTaskAction `
  -Execute "C:\ProgramData\opensourcebackup\opensourcebackup-agent.exe" `
  -Argument "run"
$trigger  = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit 0 -RestartCount 3
$principal = New-ScheduledTaskPrincipal -UserId "$env:USERDOMAIN\$env:USERNAME" -RunLevel Highest

Register-ScheduledTask -TaskName "OpenSourceBackupAgent" `
  -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force

Start-ScheduledTask -TaskName "OpenSourceBackupAgent"
```

**Vorteil:** Läuft im Hintergrund, hat Zugriff auf Netzlaufwerke des angemeldeten Benutzers.

---

## 16. TypeScript Build-Fehler: unused variables

**Häufige Fehler:**

| Fehler | Ursache | Fix |
|---|---|---|
| `'VERSION' is declared but never read` | Konstante nach Refactoring nicht mehr verwendet | Entfernen |
| `'scoreResult' is declared but never read` | Alte Berechnung durch Backend-Endpoint ersetzt | Entfernen |
| `'size' is declared but never read` | Parameter in Funktion nicht genutzt | `_size` benennen |
| `'useEffect' is declared but never read` | Import nach Refactoring übrig | Aus Import entfernen |
| `Property 'Type' does not exist on type 'BackupJob'` | Neues DB-Feld nicht im TypeScript-Interface | Interface in `api.ts` erweitern |

**Regel:** Nach jedem Feature-Block `npx tsc --noEmit` lokal ausführen, nicht nur beim Build auf Proxmox.

---

## 17. Vite base-Pfad: Assets laden nicht unter /ui/

**Symptom:**  
Dashboard lädt (kein Redirect-Loop mehr), aber ist weiß. Browser-Console zeigt CSS/JS-Fehler. `<head>` ist leer.

**Ursache:**  
Vite baut ohne `base`-Einstellung alle Asset-Pfade als absolute URLs (`/assets/...`). Der Server mappt aber nur `/ui/` — daher können `/assets/index.js` nicht gefunden werden.

**Lösung:**  
`web/vite.config.ts`:
```ts
export default defineConfig({
  plugins: [react()],
  base: '/ui/',   // ← Alle Assets werden als /ui/assets/... referenziert
})
```

Danach muss das Web UI neu gebaut werden:
```bash
cd web && npm run build && cp -r dist/. /opt/opensourcebackup/web-ui/
```

---

## Allgemeine Deployment-Checkliste (Proxmox)

Nach jeder Aktualisierung diese Reihenfolge einhalten:

```bash
# 1. Code holen
cd /tmp/osb-build && git pull   # falls Verzeichnis fehlt: git clone ... /tmp/osb-build

# 2. Migrationen zuerst ausführen
DB_PASS=$(cat /etc/opensourcebackup/.db_password)
migrate -path migrations \
  -database "postgres://opensourcebackup:${DB_PASS}@127.0.0.1:5432/opensourcebackup?sslmode=disable" up

# 3. Server neu bauen
export PATH="/usr/local/go/bin:$PATH"
go build -o /opt/opensourcebackup/opensourcebackup-server ./cmd/control-plane/...

# 4. Web UI neu bauen
cd web && npm run build && cp -r dist/. /opt/opensourcebackup/web-ui/ && cd ..

# 5. Dienst neu starten
systemctl restart opensourcebackup
systemctl status opensourcebackup --no-pager | head -8
```

---

---

## 13. Agent Windows: sc.exe create haengt sich auf

**Symptom:**  
Windows-Dienst wird mit `sc.exe create` angelegt (`Stopped`, StartType `Automatic`), hängt aber beim Start — `Start-Service` kehrt nicht zurück.

**Ursache:**  
Das Agent-Binary ist eine normale Console-Applikation (Go). Es implementiert **nicht** die Windows Service Control API (`svc.Run()`). Der SCM wartet auf das Service-Ready-Signal — das niemals kommt.

**Loesung — Task Scheduler (empfohlen):**

```powershell
# Als Administrator:
schtasks /Create `
  /TN "OpenSourceBackupAgent" `
  /TR "C:\ProgramData\OpenSourceBackup\opensourcebackup-agent.exe" `
  /SC ONSTART `
  /RU SYSTEM `
  /RL HIGHEST `
  /F

schtasks /Run /TN "OpenSourceBackupAgent"
```

**Loesung — HKCU-Autostart (kein Admin, startet bei Login):**

```powershell
Set-ItemProperty `
  -Path "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" `
  -Name "OpenSourceBackupAgent" `
  -Value "C:\ProgramData\OpenSourceBackup\opensourcebackup-agent.exe"
```

**Umgebungsvariablen systemweit setzen (einmalig als Admin):**

```powershell
[System.Environment]::SetEnvironmentVariable("OSB_SERVER_URL","http://<server-ip>:8080","Machine")
[System.Environment]::SetEnvironmentVariable("OSB_TOKEN_FILE","C:\ProgramData\OpenSourceBackup\agent-token","Machine")
```

> Falls zukuenftig nativer Service-Support gewuenscht wird: `golang.org/x/sys/windows/svc` im Agent einbinden.

---

*Dieses Dokument wird bei neuen bekannten Problemen erweitert.*
