# OpenSourceBackup — Projekt-Kontext für Claude

## Stack
- Go 1.22+ Control Plane (`cmd/control-plane`) + Agent (`cmd/agent`)
- PostgreSQL 16 + Redis via Docker
- React 18 + TypeScript + Vite (`web/`) — `base: '/ui/'`
- Restic Backup Engine

## Dev-Workflow
```bash
# Backend
go test ./...
go build ./...

# Frontend
cd web && npm run build
npx tsc --noEmit   # TypeScript-Check

# Cross-compile für Proxmox (Linux)
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -ldflags="-s -w" -o osb-server-linux ./cmd/control-plane
```

## Design-System
**IMMER** die Design-Branding-Datei lesen bevor UI gebaut wird:
`C:\Users\Admin\.claude\projects\C--Users-Admin-Documents-Unternemens-Struktur-development-projects-OpensourceBackup\memory\design_branding.md`

Kurzregeln:
- Dark Theme default — CSS-Variablen aus `web/src/index.css`
- Akzentfarbe: `--accent: #89BD28` (OSB-Grün)
- Cards: `linear-gradient(180deg, rgba(21,28,46,0.95), rgba(10,15,27,0.95))`
- Styles als `const s: Record<string, React.CSSProperties>` — keine CSS-Klassen außer `.dash-card`
- Leere Zustände: Icon (opacity 0.3) + Titel + Subtext
- Keine Fake-Daten, keine `any`-Flut, kein `borderOpacity`

## Kritische Architektur-Entscheidungen
- `BrowserRouter` mit `basename="/ui"` (Pflicht!)
- API-Calls relativ (kein `localhost:8080` hardcoded) — `BASE = import.meta.env.VITE_API_URL || ''`
- CSRF Double-Submit Cookie (`X-CSRF-Token` bei POST/PUT/DELETE)
- Auth: `authEnabled = adminEmail != "" && adminPass != ""` (BEIDE müssen gesetzt sein)
- Agent: `CONTROL_PLANE_URL` (nicht `OSB_SERVER_URL`), `AGENT_TOKEN_FILE`
- `sc.exe create` → funktioniert NICHT für Agent (kein Windows Service API) → Task Scheduler

## Deployment
- Server: `192.168.0.72:8080` (LXC auf Proxmox)
- SSH-Key: `C:\Users\Admin\.ssh\id_ed25519` (noch nicht auf Server eingetragen)
- Web-Deploy: `docker compose cp dist/. osb-web:/usr/share/nginx/html/ui/`
- Kein Root-Passwort bekannt → Proxmox-Console oder `pct enter <CTID>`
- Details: `memory/feedback_infrastructure.md`

## Kein Push ohne
- [ ] `go test ./...` grün
- [ ] `npx tsc --noEmit` clean
- [ ] Keine Secrets / Pfade im Code
- [ ] `docs/security/` niemals pushen (in `.gitignore`)
