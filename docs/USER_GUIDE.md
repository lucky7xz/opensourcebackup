# User Guide — OpensourceBackup

> Anleitung für den Betrieb und die Nutzung des OpensourceBackup Control Plane.
> Stand: B1–B9.7 — REST API + Backup-Agent verfügbar.

---

## Was ist OpensourceBackup?

OpensourceBackup sichert Dateien, Ordner und Datenbanken auf deinen Servern und Clients.
Es besteht aus zwei Teilen:

```
Control Plane (läuft zentral)
  → verwaltet Systeme, Policies, Jobs, Snapshots
  → plant Backups (Cron-Scheduler)
  → überwacht ob Backups wirklich laufen (Dead-Man's-Switch)

Agent (läuft auf jedem Zielsystem)
  → holt Jobs von der Control Plane
  → führt restic backup aus
  → meldet Ergebnis zurück
```

**Was du sichern kannst:**
- Dateien & Ordner: `/home`, `/etc`, `/var/www`, `C:\Users\` → via Restic
- PostgreSQL-Datenbanken → via pgBackRest
- Komplette Systeme → via Restic
- Kubernetes-Cluster → via Velero

---

## Starten

### Control Plane

```bash
git clone https://github.com/cerberus8484/opensourcebackup.git
cd opensourcebackup

make dev-up        # PostgreSQL + Redis starten
make migrate-up    # Datenbank einrichten
make run           # → http://localhost:8080

curl http://localhost:8080/health
# → {"status":"ok"}
```

### Agent auf einem Zielsystem einrichten

```bash
# Schritt 1: System in der Control Plane anlegen
curl -X POST http://localhost:8080/v1/systems \
  -H "Content-Type: application/json" \
  -d '{"Hostname": "mein-server", "RiskClass": "standard"}'
# → {"ID": "uuid-des-systems", ...}

# Schritt 2: Enrollment-Token erzeugen (gilt 30 Minuten)
curl -X POST http://localhost:8080/v1/systems/{uuid-des-systems}/enrollment-token
# → {"token": "Xk3mNp...", "expires_at": "..."}

# Schritt 3: Agent starten — enrollt sich automatisch
CONTROL_PLANE_URL=http://localhost:8080 \
ENROLLMENT_TOKEN=Xk3mNp... \
RESTIC_PASSWORD=geheimes-passwort \
RESTIC_REPO=s3:mein-bucket/backups/mein-server \
./agent

# → Agent speichert Token in data/agent-token
# → Agent pollt alle 30s nach Jobs
```

**Nach dem ersten Enrollment** genügt:
```bash
CONTROL_PLANE_URL=http://localhost:8080 \
RESTIC_PASSWORD=geheimes-passwort \
RESTIC_REPO=s3:mein-bucket/backups \
./agent
# → liest Token aus data/agent-token
```

---

## API-Referenz

Base-URL: `http://localhost:8080`

---

### Systeme `/v1/systems`

Ein System = ein zu sicherndes Gerät.

```bash
# Anlegen
curl -X POST http://localhost:8080/v1/systems \
  -H "Content-Type: application/json" \
  -d '{"Hostname":"web-01","OS":"Ubuntu 22.04","RiskClass":"critical","Tags":{"env":"prod"}}'

# Alle abrufen
curl http://localhost:8080/v1/systems

# Einzeln abrufen / aktualisieren / löschen
curl http://localhost:8080/v1/systems/{id}
curl -X PUT http://localhost:8080/v1/systems/{id} -d '...'
curl -X DELETE http://localhost:8080/v1/systems/{id}
```

### Repositories `/v1/repositories`

Ein Repository = Backup-Speicherziel (S3, MinIO, lokales Laufwerk).

```bash
curl -X POST http://localhost:8080/v1/repositories \
  -H "Content-Type: application/json" \
  -d '{
    "Type": "restic",
    "Location": "s3:mein-bucket/backups/web-01",
    "EncryptionMode": "aes256",
    "ObjectLockEnabled": true
  }'
```

| Feld | Pflicht | Beschreibung |
|---|---|---|
| `Type` | ✅ | `restic`, `borg`, `pgbackrest`, `velero` |
| `Location` | ✅ | Pfad oder URL |
| `ObjectLockEnabled` | — | WORM-Schutz gegen Ransomware |

### Policies `/v1/policies`

Eine Policy = was wird wann wie gesichert, und wohin.

```bash
curl -X POST http://localhost:8080/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "nightly-full",
    "Engine": "restic",
    "RepositoryID": "uuid-des-repositories",
    "Includes": ["/home", "/etc"],
    "Excludes": ["/home/*/.cache"],
    "Schedule": "0 2 * * *",
    "Retention": {"daily": 7, "weekly": 4, "monthly": 12}
  }'
```

**Wichtig:** `RepositoryID` muss gesetzt sein, sonst schlägt der Backup-Job fehl.

**Cron-Beispiele:**

| Schedule | Bedeutung |
|---|---|
| `0 2 * * *` | Täglich 02:00 Uhr |
| `0 2 * * 0` | Wöchentlich, Sonntags |
| `0 */6 * * *` | Alle 6 Stunden |

### Enrollment-Token `/v1/systems/{id}/enrollment-token`

```bash
# Einmaligen Token für einen Agent erzeugen (gilt 30 Minuten)
curl -X POST http://localhost:8080/v1/systems/{id}/enrollment-token
# → {"token": "...", "system_id": "...", "expires_at": "..."}
```

---

### Agent-Routen `/v1/agent/*`

Diese Routen sind **nur für Agents** — immer mit Bearer-Token:

```bash
Authorization: Bearer <agent-token>
```

| Route | Beschreibung |
|---|---|
| `POST /v1/agent/enroll` | Enrollment-Token → Agent-Token tauschen |
| `GET /v1/agent/jobs` | Pending Jobs für dieses System |
| `PUT /v1/agent/jobs/{id}/start` | Job als "running" markieren |
| `PUT /v1/agent/jobs/{id}/complete` | Backup erfolgreich, Snapshot registrieren |
| `PUT /v1/agent/jobs/{id}/fail` | Backup fehlgeschlagen |

Ein Agent sieht **nur** seine eigenen Jobs — nie Jobs anderer Systeme.

---

### Jobs `/v1/jobs`

```bash
# Alle Jobs
curl http://localhost:8080/v1/jobs

# Pending Jobs für ein System (wie der Agent es nutzt)
curl "http://localhost:8080/v1/jobs?system_id={id}&status=pending"
```

**Job-Status:**

| Status | Bedeutung |
|---|---|
| `pending` | Wartet auf Agent |
| `running` | Agent führt Backup aus |
| `success` | Erfolgreich |
| `failed` | Fehlgeschlagen |

### Snapshots `/v1/snapshots`

```bash
curl http://localhost:8080/v1/snapshots
curl http://localhost:8080/v1/snapshots/{id}
```

---

## Scheduler & Monitoring

**Automatische Job-Erstellung:** Der Scheduler liest beim Start alle Policies mit Cron-Schedule
und erstellt automatisch `pending`-Jobs zur richtigen Zeit.

**Dead-Man's-Switch:** Alle 5 Minuten prüft der Scheduler ob alle Jobs termingerecht gelaufen sind.
Wenn der letzte Job älter als `Intervall × 1.5` ist:

```json
{"level":"WARN","msg":"dead-man: overdue job detected",
 "policy_name":"nightly-full","last_job_at":"2026-05-31T02:00:00Z"}
```

---

## HTTP-Statuscodes

| Code | Bedeutung |
|---|---|
| `200` | Erfolgreich |
| `201` | Ressource angelegt |
| `204` | Gelöscht |
| `400` | Ungültige Eingabe |
| `401` | Kein oder ungültiger Token |
| `404` | Nicht gefunden (auch: falsches System bei Agent-Routen) |
| `413` | Request Body > 1 MB |
| `503` | DB nicht erreichbar oder Timeout |

---

## Häufige Fragen

**Agent meldet "re-enrollment required" — was tun?**
Der Agent-Token wurde revoked oder ist ungültig. Neuen Enrollment-Token erstellen und
Agent neu starten mit `ENROLLMENT_TOKEN=...`.

**Backup schlägt fehl mit "policy has no repository configured"?**
Policy hat keine `RepositoryID`. Policy aktualisieren:
```bash
curl -X PUT http://localhost:8080/v1/policies/{id} \
  -d '{"...", "RepositoryID": "uuid-des-repositories"}'
```

**DB-Verbindung prüfen:**
```bash
curl http://localhost:8080/health
# {"status":"ok"} → OK
# HTTP 503        → DB nicht erreichbar
```

**Alles zurücksetzen:**
```bash
make migrate-down && make migrate-up
```

---

## Sicherheitshinweise

- Agent-Token liegt in `data/agent-token` mit Rechten `0600` — nur Owner lesbar
- `RESTIC_PASSWORD` niemals in Logs ausgeben
- `DATABASE_URL` enthält Credentials — niemals committen
- Enrollment-Token gilt nur 30 Minuten und kann nur einmal verwendet werden
- Control Plane setzt automatisch Security Headers auf jede Response
