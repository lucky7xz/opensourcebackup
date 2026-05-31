# User Guide — OpensourceBackup

> Stand: B1–B16 — Control Plane, Agent, Web-Dashboard.

---

## Was ist OpensourceBackup?

OpensourceBackup sichert Dateien, Ordner und Datenbanken auf deinen Servern und Clients — zentral verwaltet, automatisch überwacht.

```
Control Plane + Web-UI  → verwaltet Systeme, Policies, Jobs, Snapshots
Agent (auf Zielsystem)  → führt restic backup aus, meldet zurück
```

**Kern-Frage des Dashboards:** *Sind meine Systeme gesichert — und wurde ein Restore getestet?*

---

## Starten

```bash
# 1. Control Plane + Datenbank
make dev-up && make migrate-up && make run

# 2. Web-UI
cd web && npm install && npm run dev
# → http://localhost:5173

# 3. Gesundheit prüfen
curl http://localhost:8080/health  # → {"status":"ok"}
```

---

## Erster Backup in 4 Schritten

### Schritt 1 — System registrieren
**Systems-Seite → (kommt in B_FORM)**  
Aktuell per API:
```powershell
Invoke-WebRequest "http://localhost:8080/v1/systems" -Method POST `
  -ContentType "application/json" `
  -Body '{"Hostname":"mein-server","RiskClass":"standard"}' `
  -UseBasicParsing
```

### Schritt 2 — Repository anlegen
**Repositories-Seite** (Create-Formular kommt in B_FORM)
```powershell
Invoke-WebRequest "http://localhost:8080/v1/repositories" -Method POST `
  -ContentType "application/json" `
  -Body '{"Type":"restic","Location":"C:/tmp/backup-repo"}' `
  -UseBasicParsing
```

### Schritt 3 — Policy erstellen
**Policies-Seite → „+ New Policy"**

| Feld | Beschreibung |
|---|---|
| Name | z.B. `nightly-documents` |
| Engine | restic / borg / pgbackrest / velero |
| Repository | Dropdown aus vorhandenen Repos |
| Include Paths | Ordner zum Sichern, z.B. `C:/Users/Admin/Documents` |
| Exclude Paths | Optional: `C:/Users/Admin/AppData` |
| Schedule | Preset oder eigener Cron-Ausdruck |
| Retention | Wie viele Snapshots behalten (Daily / Weekly / Monthly) |

### Schritt 4 — Agent installieren
**Agents-Seite → Install Agent Wizard**

1. System aus Liste wählen
2. Platform wählen (Windows / Linux)
3. Repository-Pfad + Passwort eingeben
4. Fertigen Installationsbefehl kopieren und auf Zielsystem ausführen

Der Agent enrollt sich automatisch und startet den Backup-Zyklus.

---

## Web-UI Übersicht

| Seite | URL | Was du siehst |
|---|---|---|
| Dashboard | `/` | Health-Cards, Restore-Status, Recent Jobs |
| Systems | `/systems` | Alle Systeme, Last Backup, ▶ Run Backup |
| Agents | `/agents` | Install-Wizard, Connected Systems, Remove |
| Policies | `/policies` | Backup-Regeln, + New Policy, 🗑 Delete |
| Jobs | `/jobs` | Jobs mit Filter, + New Job, 🗑 Delete (pending/failed) |
| Snapshots | `/snapshots` | Alle Snapshots, Restore-Test-Status |
| Restore Tests | `/restore-tests` | Kommt in B13/B14 |
| Repositories | `/repositories` | Storage-Ziele |

---

## Agent verwalten

### Agent herunterladen
```powershell
# Windows
Invoke-WebRequest "http://localhost:8080/downloads/agent/v0.1.0/windows-amd64" `
  -OutFile opensourcebackup-agent.exe

# Linux
curl -fsSL http://localhost:8080/downloads/agent/v0.1.0/linux-amd64 `
  -o opensourcebackup-agent && chmod +x opensourcebackup-agent
```

### Agent starten (nach Enrollment)
```powershell
$env:CONTROL_PLANE_URL = "http://localhost:8080"
$env:RESTIC_PASSWORD   = "<passwort>"
$env:RESTIC_REPO       = "<pfad-oder-url>"
.\opensourcebackup-agent.exe
```

### Agent stoppen
```powershell
Stop-Process -Name "opensourcebackup-agent" -Force
```

### Agent entfernen
**Agents-Seite** oder **Systems-Seite** → 🗑 Remove → Bestätigung

Der Agent stoppt beim nächsten Poll (max. 30s) mit 401.

---

## Backup manuell auslösen

**Jobs-Seite → „+ New Job"** → System + Policy wählen → Run Backup

Oder auf der **Systems-Seite** → ▶ Run Backup in der jeweiligen Zeile.

---

## Policy-Pfade konfigurieren

**Policies-Seite → „+ New Policy"**

Include-Pfade:
```
C:/Users/Admin/Documents
C:/Users/Admin/Desktop
C:/ProgramData/myapp
```

Exclude-Pfade (optional):
```
C:/Users/Admin/AppData
C:/Users/Admin/Documents/temp
```

Nach dem Erstellen: Job manuell oder per Schedule auslösen.

---

## Scheduler

Alle Policies mit einem Cron-Schedule werden beim Start der Control Plane geladen.
**Neue Policy → Control Plane neu starten damit der Schedule aktiv wird.**

**Dead-Man's-Switch:** Alle 5 Min prüft der Scheduler ob Jobs termingerecht liefen:
```json
{"level":"WARN","msg":"dead-man: overdue job detected","policy_name":"nightly"}
```

---

## HTTP-Statuscodes

| Code | Bedeutung |
|---|---|
| 200 / 201 / 204 | OK / Angelegt / Gelöscht |
| 400 | Ungültige Eingabe (z.B. fehlende Pflichtfelder) |
| 401 | Kein oder ungültiger Bearer-Token |
| 404 | Nicht gefunden |
| 413 | Request Body > 1 MB |
| 503 | DB nicht erreichbar oder Timeout |

---

## Sicherheitshinweise

- Agent-Token in `data/agent-token` mit Rechten `0600`
- Enrollment-Token gilt nur 30 Minuten, nur einmal verwendbar
- `RESTIC_PASSWORD` und `DATABASE_URL` nie in Logs ausgeben
- System löschen → alle Tokens werden automatisch mitgelöscht (CASCADE)
