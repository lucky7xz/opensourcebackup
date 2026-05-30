# User Guide — OpensourceBackup

> Anleitung für den Betrieb und die Nutzung des OpensourceBackup Control Plane.
> Stand: B1–B7 — REST API verfügbar, Auth kommt in B9.

---

## Was ist OpensourceBackup?

OpensourceBackup ist eine **Backup Control Plane** — eine zentrale Plattform, die
Backup-Jobs auf vielen Systemen gleichzeitig verwaltet, überwacht und koordiniert.

```
Deine Systeme → Backup-Agent → Control Plane → Storage
                                     ↑
                               Du steuerst hier
```

**Was du damit tust:**
- Systeme registrieren (Server, VMs, Datenbanken)
- Backup-Repositories definieren (S3, MinIO, ZFS)
- Policies anlegen (was wird wann wie gesichert)
- Jobs überwachen
- Snapshots verwalten

---

## Starten

### Lokal (Entwicklung)

```bash
# Voraussetzungen: Docker, Go 1.22+
git clone https://github.com/cerberus8484/opensourcebackup.git
cd opensourcebackup

make dev-up        # PostgreSQL + Redis starten
make migrate-up    # Datenbanktabellen anlegen
make run           # Control Plane starten → http://localhost:8080
```

### Gesundheitsprüfung

```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```

---

## API-Referenz

Base-URL: `http://localhost:8080` (Entwicklung)

Alle Anfragen und Antworten sind JSON. Alle IDs sind UUIDs.

---

### Systeme (`/v1/systems`)

Ein System ist ein zu sicherndes Gerät: Server, VM, Datenbank, Endgerät.

#### System anlegen

```bash
curl -X POST http://localhost:8080/v1/systems \
  -H "Content-Type: application/json" \
  -d '{
    "Hostname": "web-server-01.example.com",
    "OS": "Ubuntu 22.04",
    "RiskClass": "critical",
    "Tags": {"env": "prod", "team": "backend"}
  }'
```

**Felder:**

| Feld | Pflicht | Beschreibung |
|---|---|---|
| `Hostname` | ✅ | Eindeutiger Hostname |
| `OS` | — | Betriebssystem |
| `AgentVersion` | — | Version des installierten Agents |
| `RiskClass` | — | `standard` (Default) oder `critical` |
| `Tags` | — | Freie Key-Value-Metadaten (JSON) |

#### Alle Systeme abrufen

```bash
curl http://localhost:8080/v1/systems
```

#### System abrufen

```bash
curl http://localhost:8080/v1/systems/{id}
```

#### System aktualisieren

```bash
curl -X PUT http://localhost:8080/v1/systems/{id} \
  -H "Content-Type: application/json" \
  -d '{"Hostname": "web-server-01.example.com", "RiskClass": "standard"}'
```

#### System löschen

```bash
curl -X DELETE http://localhost:8080/v1/systems/{id}
# → HTTP 204 No Content
```

---

### Repositories (`/v1/repositories`)

Ein Repository ist ein Backup-Speicherziel (S3-Bucket, MinIO, ZFS-Dataset).

#### Repository anlegen

```bash
curl -X POST http://localhost:8080/v1/repositories \
  -H "Content-Type: application/json" \
  -d '{
    "Type": "restic",
    "Location": "s3:mein-bucket/backups/web-server-01",
    "EncryptionMode": "aes256",
    "ObjectLockEnabled": true
  }'
```

**Felder:**

| Feld | Pflicht | Beschreibung |
|---|---|---|
| `Type` | ✅ | Engine: `restic`, `borg`, `pgbackrest`, `velero` |
| `Location` | ✅ | Pfad oder URL zum Storage |
| `EncryptionMode` | — | Verschlüsselungsmodus |
| `ObjectLockEnabled` | — | WORM-Schutz (Ransomware-Schutz) |

---

### Policies (`/v1/policies`)

Eine Policy definiert: was wird gesichert, wann, mit welcher Engine, wie lange aufbewahrt.

#### Policy anlegen

```bash
curl -X POST http://localhost:8080/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "nightly-full-backup",
    "Engine": "restic",
    "Includes": ["/home", "/etc", "/var/www"],
    "Excludes": ["/home/*/.cache", "/var/www/tmp"],
    "Schedule": "0 2 * * *",
    "Retention": {"daily": 7, "weekly": 4, "monthly": 12}
  }'
```

**Felder:**

| Feld | Pflicht | Beschreibung |
|---|---|---|
| `Name` | ✅ | Eindeutiger Name |
| `Engine` | ✅ | `restic`, `borg`, `pgbackrest`, `velero` |
| `Includes` | — | Zu sichernde Pfade |
| `Excludes` | — | Ausgeschlossene Pfade |
| `Schedule` | — | Cron-Ausdruck (`0 2 * * *` = täglich 02:00 Uhr) |
| `Retention` | — | Aufbewahrungsregeln als JSON |
| `PreHooks` | — | Kommandos vor dem Backup |
| `PostHooks` | — | Kommandos nach dem Backup |

**Cron-Beispiele:**

| Schedule | Bedeutung |
|---|---|
| `0 2 * * *` | Täglich um 02:00 Uhr |
| `0 2 * * 0` | Wöchentlich, Sonntags 02:00 Uhr |
| `0 */6 * * *` | Alle 6 Stunden |
| `@daily` | Einmal täglich |

---

### Jobs (`/v1/jobs`)

Ein Job ist eine ausgeführte oder geplante Backup-Instanz.

#### Job manuell anlegen (pending)

```bash
curl -X POST http://localhost:8080/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "SystemID": "uuid-des-systems",
    "PolicyID": "uuid-der-policy",
    "Status": "pending"
  }'
```

**Job-Status:**

| Status | Bedeutung |
|---|---|
| `pending` | Warte auf Agent |
| `running` | Agent führt Backup aus |
| `success` | Backup erfolgreich |
| `failed` | Backup fehlgeschlagen |
| `warning` | Backup mit Warnungen |

#### Alle Jobs abrufen

```bash
curl http://localhost:8080/v1/jobs
```

#### Job-Status aktualisieren

```bash
curl -X PUT http://localhost:8080/v1/jobs/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "Status": "success",
    "BytesScanned": 10737418240,
    "BytesUploaded": 1073741824
  }'
```

---

### Snapshots (`/v1/snapshots`)

Ein Snapshot ist das Ergebnis eines erfolgreichen Backup-Jobs — die Referenz auf die
tatsächlich gesicherten Daten in der Backup-Engine.

#### Snapshot anlegen

```bash
curl -X POST http://localhost:8080/v1/snapshots \
  -H "Content-Type: application/json" \
  -d '{
    "JobID": "uuid-des-jobs",
    "RepositoryID": "uuid-des-repositories",
    "EngineSnapshotID": "abc123def456",
    "Hostname": "web-server-01",
    "Paths": ["/home", "/etc"],
    "ChecksumStatus": "verified"
  }'
```

---

## Scheduler & Dead-Man's-Switch

Der Scheduler lädt beim Start alle Policies mit einem Cron-Schedule und plant
automatisch Backup-Jobs.

**Dead-Man's-Switch:** Alle 5 Minuten prüft der Scheduler ob alle Policies
fristgerecht ausgeführt wurden. Wenn der letzte Job älter als `Intervall × 1.5` ist,
wird eine Warnung geloggt:

```json
{"level":"WARN","msg":"dead-man: overdue job detected",
 "policy_id":"...","policy_name":"nightly-full-backup",
 "last_job_at":"2024-01-01T02:00:00Z","overdue_since":"..."}
```

---

## HTTP-Statuscodes

| Code | Bedeutung |
|---|---|
| `200 OK` | Erfolgreich |
| `201 Created` | Ressource angelegt |
| `204 No Content` | Gelöscht |
| `400 Bad Request` | Ungültige Eingabe (Fehlermeldung im Body) |
| `404 Not Found` | Ressource nicht gefunden |
| `413 Request Entity Too Large` | Body > 1 MB |
| `503 Service Unavailable` | Timeout oder DB nicht erreichbar |

Fehler-Body:
```json
{"error": "hostname is required"}
```

---

## Häufige Fragen

**Warum muss ich mich nicht anmelden?**
Auth kommt in B9. In der aktuellen Version (B1–B7) ist die API noch offen.
**Nicht für Produktionseinsatz verwenden.**

**Wie überprüfe ich ob die DB verbunden ist?**
```bash
curl http://localhost:8080/health
# → {"status":"ok"}          DB erreichbar
# → HTTP 503                 DB nicht erreichbar
```

**Wie lese ich die Logs?**
Die Control Plane loggt strukturiertes JSON:
```bash
make run 2>&1 | jq .
```

**Wie setze ich alles zurück?**
```bash
make migrate-down  # Alle Tabellen löschen
make migrate-up    # Neu anlegen
```

---

## Datenschutz & Sicherheitshinweise

- Keine Produktionsdaten in der aktuellen Version speichern (keine Auth)
- `DATABASE_URL` enthält Credentials — niemals in Logs ausgeben
- `.env.local` niemals committen — ist in `.gitignore`
- Backup-Repository-Credentials (S3-Keys etc.) kommen in HashiCorp Vault (B9+)
