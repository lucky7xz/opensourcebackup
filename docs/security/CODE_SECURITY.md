# Code Security

> Verbindliche Sicherheitsregeln für OpensourceBackup.
> Wird bei jedem Feature das Auth, API, Input, Secrets, Deps, Restore oder Container berührt geprüft.
>
> Verwandte Dokumente:
> - [Lint-Strategie](../quality/lint-strategy.md) — gosec wandert in Schicht 1 sobald Auth steht
> - [Clean Code & Wertesystem](../developer-guide/CLEAN_CODE.md) — Korrektheit als Grundwert
> - [Architektur](../architecture/ARCHITECTURE.md) — mTLS, RBAC, Vault/SOPS
> - [Developer Guide](../developer-guide/DEVELOPER_GUIDE.md) — Sicherheitsregeln im Dev-Workflow

---

## Status-Legende

| Symbol | Bedeutung |
|---|---|
| ✅ | Implementiert |
| 🔧 | Teilweise / vorbereitet |
| ❌ | Fehlt — **blockiert Produktion** |
| 📋 | Geplant — kommt in B8–B13 |

---

## 1. Authentifizierung & Autorisierung

### Passwort-Hashing
**Status: 📋 (B9)**

- Niemals Klartext-Passwörter speichern
- Algorithmus: **bcrypt** (cost ≥ 12) oder **Argon2id**
- MD5, SHA-1, SHA-256 für Passwörter: **absolut verboten**

```go
// ✅ bcrypt
hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)

// ❌ Verboten
hash := sha256.Sum256([]byte(password))
```

### Token-basierte Authentifizierung
**Status: ❌ — blockiert Produktion**

- JWT für Benutzer: kurzlebig (15–60 Min), signiert RS256 oder HS256
- mTLS für Agent ↔ Control Plane: TLS-Client-Zertifikate, keine Passwörter
- HttpOnly-Cookies für Web-Clients: verhindert XSS-Token-Diebstahl
- Refresh-Token: langlebig, rotierend, in DB invalidierbar

```go
const (
    accessTokenTTL  = 15 * time.Minute
    refreshTokenTTL = 7 * 24 * time.Hour
)
```

### Session-Management
**Status: ❌ — blockiert Produktion**

- Sessions bei Logout invalidieren (DB-Revocation oder Token-Blacklist)
- SameSite=Strict für Auth-Cookies
- Keine Session-IDs in URLs

### RBAC — Role-Based Access Control
**Status: 📋 (B9)**

- Rollen: `admin`, `operator`, `viewer`, `agent`
- Restore-Rechte **getrennt** von Backup-Rechten (laut Architektur)
- Least Privilege: jeder Principal bekommt nur das Minimum

### Authorization-Negativtests
**Status: 📋 (B9) — Pflicht sobald RBAC steht**

Nicht nur testen dass etwas funktioniert — auch testen dass es **nicht** funktioniert:

```
- viewer darf keine Policy ändern → 403
- operator darf keine Admins verwalten → 403
- agent darf keine fremden Jobs lesen → 403
- ungültiger Token → 401
- gültiger Token ohne Rechte → 403
- gelöschter Agent kann keine Jobs mehr abrufen → 401
- abgelaufener Token → 401
```

### MFA
**Status: 📋 — optional für v1, Pflicht für Admin-Accounts**

- TOTP via Authenticator-App
- Backup-Codes bei Einrichtung generieren

---

## 2. Agent-Sicherheit

**Status: 📋 (B9/B10) — kritisch für das Projekt**

Agent-Enrollment-Sicherheit ist ein Hochrisiko-Bereich:

```
- Enrollment Token ist einmalig verwendbar (OTP-Semantik)
- Enrollment Token hat kurze TTL: 10–30 Minuten
- Nach Enrollment: Agent bekommt eigenes Zertifikat / eigene Identität
- Zertifikate können widerrufen werden (CRL oder OCSP)
- Agent darf nur Jobs für das eigene System abrufen
- Agent darf nur allowgelistete Kommandos ausführen
- Control Plane validiert jede Agent-Antwort
- Kein Agent darf Jobs anderer Systeme sehen oder beeinflussen
```

```go
// Agent-Job-Isolation — Pflicht
// GET /v1/jobs?system_id={eigene_ID} — nur eigene Jobs sichtbar
// Jede Abfrage: system_id aus dem Agent-Zertifikat extrahieren, nicht aus dem Request
```

---

## 3. Datenvalidierung & Bereinigung

### Whitelist-Validation statt Blacklist
**Status: 🔧 — Pflichtfelder geprüft, Formate noch nicht**

```go
// ✅ Whitelist: erlaubte Werte explizit
var validEngines = map[string]bool{
    "restic": true, "borg": true, "pgbackrest": true, "velero": true,
}
if !validEngines[p.Engine] {
    writeError(w, http.StatusBadRequest, "unsupported engine")
    return
}

// ✅ Längen-Limits
if len(s.Hostname) == 0 || len(s.Hostname) > 253 {
    writeError(w, http.StatusBadRequest, "hostname: 1–253 chars required")
    return
}

// ✅ Cron-Ausdruck validieren vor Speicherung
if _, err := cron.ParseStandard(schedule); err != nil {
    writeError(w, http.StatusBadRequest, "invalid cron expression")
    return
}
```

### Pagination-Limits
**Status: 📋 (B8)**

```go
// Ohne Limit: ein Request kann alle Daten laden
const maxPageSize = 100
if limit > maxPageSize {
    limit = maxPageSize
}
```

### SQL-Injection-Schutz
**Status: ✅ — vollständig**

Alle Queries via pgx mit `$1, $2` Platzhaltern. Kein String-Concat mit User-Input.
Automatisch geprüft: `staticcheck` (Schicht 1)

### Script-Injection / XSS
**Status: 📋 — relevant wenn Web-UI kommt**

- Alle User-Daten in HTML escapen
- Content Security Policy Header setzen
- React/Vue escapen automatisch — kein `dangerouslySetInnerHTML`

---

## 4. API-Sicherheit & Request-Limits

### Request-Größen und DoS-Schutz
**Status: 🔧 — Timeouts vorhanden, Body-Limit fehlt**

```go
// ✅ Bereits implementiert
srv := &http.Server{
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 35 * time.Second,
}

// ❌ Fehlt noch — Pflicht für B8
func RequestLimits(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
        next.ServeHTTP(w, r)
    })
}

// Empfohlen: ReadHeaderTimeout und IdleTimeout ergänzen
srv := &http.Server{
    ReadTimeout:       10 * time.Second,
    ReadHeaderTimeout: 5 * time.Second,
    WriteTimeout:      35 * time.Second,
    IdleTimeout:       60 * time.Second,
}
```

### Rate Limiting
**Status: ❌ — fehlt**

- Alle Endpunkte: max. 100 Req/Min pro IP
- Auth-Endpunkte: max. 10 Req/Min pro IP (Brute-Force-Schutz)
- Algorithmus: Token-Bucket via `golang.org/x/time/rate`

### HTTPS / TLS
**Status: ❌ — blockiert Produktion**

- TLS 1.3 minimum, TLS 1.2 als Fallback
- mTLS zwischen Agent und Control Plane (laut Architektur)

```go
tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13}
```

### Security Headers
**Status: ❌ — fehlt**

```go
// Geplante Middleware für B7
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        next.ServeHTTP(w, r)
    })
}
```

### CORS
**Status: 📋 — relevant wenn Web-UI kommt**

- Strikte Origin-Whitelist — kein `Access-Control-Allow-Origin: *` in Produktion
- Nur explizit erlaubte Methoden und Header

### CSRF-Schutz
**Status: 📋 — relevant wenn Web-UI mit Cookies kommt**

- Anti-CSRF-Token für state-verändernde Requests
- SameSite=Strict für Auth-Cookies

---

## 5. Restore-Sicherheit

**Status: 📋 (B13) — besonders kritisch für ein Backup-Tool**

Restore ist gefährlicher als Backup: man kann damit Daten überschreiben oder Schadcode zurückholen.

```
- Restore benötigt eigene Berechtigung, getrennt von Backup-Rechten ✅ (in RBAC geplant)
- Optional: 4-Augen-Prinzip für produktive Restores (2-Personen-Freigabe)
- Restore standardmäßig in Staging-/Temp-Ziel, nicht direkt produktiv
- Restore-Plan anzeigen bevor geschrieben wird (dry-run first)
- Audit-Log für jeden Restore-Vorgang (wer, wann, wohin, welcher Snapshot)
```

### Pfad-Sicherheit bei Restore
**Status: 📋 (B13)**

```go
// ✅ Pfade normalisieren und validieren
target := filepath.Clean(userInput)
if !strings.HasPrefix(target, allowedBase) {
    return errors.New("path traversal detected")
}

// ❌ Niemals so
os.MkdirAll(userInput, 0755) // user kontrolliert den Pfad

// Weitere Regeln:
// - keine ../ Sequenzen erlauben
// - Symlinks explizit behandeln — keine blinden Follows
// - temporäre Dateien mit 0600, nicht 0666
// - keine Welt-schreibbaren Verzeichnisse
// - Dateinamen aus User-Input niemals ungeprüft übernehmen
```

---

## 6. Sensible Daten & Umgebungsvariablen

### Secrets nie im Code
**Status: ✅ — eingehalten**

```go
// ✅ Immer aus Env lesen
dsn := os.Getenv("DATABASE_URL")

// ❌ Niemals
const dbPass = "supersecret"
```

Automatisch geprüft: Privacy-Check vor Push + `gosec` G101 (Schicht 2 → Schicht 1 mit B9)

### Secrets Manager für Produktion
**Status: 📋 (Deployment)**

- HashiCorp Vault oder SOPS (laut Architektur)
- Repository-Credentials: pro System/Mandant getrennt in Vault
- Rotation: Secrets haben ein TTL, werden automatisch rotiert

### Verschlüsselung at Rest
**Status: 📋 (Deployment)**

- PostgreSQL: Disk-Verschlüsselung (LUKS oder cloud-native)
- Backup-Daten: Verschlüsselung beim Agenten — kein Vertrauen in Storage-Backend
- Keys niemals zusammen mit verschlüsselten Daten speichern

### Container-Sicherheit
**Status: 📋 (Dockerfile kommt)**

```dockerfile
# ✅ Minimales Base-Image — kein ubuntu:latest
FROM gcr.io/distroless/static-debian12

# ✅ Non-root User
USER nonroot:nonroot

# ✅ Konkrete Version — kein latest
FROM postgres:16.3-alpine
```

- `latest`-Tags verboten — immer Pin auf konkrete Version
- Image-Scans via Trivy oder Grype in CI
- Container-Images signieren via Cosign

---

## 7. Fehlerbehandlung & Audit-Logging

### Generische Fehlermeldungen nach außen
**Status: ✅ — eingehalten**

```go
// ✅ Nach außen: generisch
writeError(w, http.StatusInternalServerError, "internal error")

// ✅ Intern: vollständiger Kontext
h.log.Error("db query failed", "error", err, "op", "GetByID", "id", id)

// ❌ Niemals Stack-Traces oder DB-Details nach außen
writeError(w, 500, err.Error())
```

### Produktionsmodus-Trennung
**Status: 🔧 — nicht explizit erzwungen**

```
- DEBUG=false in Produktion
- kein pprof öffentlich erreichbar
- keine Testdaten / Demo-Accounts in Produktion
- Service bindet nicht automatisch auf 0.0.0.0
- Admin-API nicht öffentlich erreichbar
- keine internen Pfade oder Versionen in Response-Headers
```

### Audit-Logging
**Status: 🔧 — Basis-Logging vorhanden, Audit-Events fehlen**

Normale Logs ≠ Audit-Logs. Audit-Logs sind append-only, nicht manipulierbar, für Compliance.

**Pflicht-Audit-Events:**

```
Authentifizierung:
  - Login erfolgreich / fehlgeschlagen (mit IP)
  - Token erstellt / widerrufen
  - Agent enrolled / revoked

Konfigurationsänderungen:
  - Policy erstellt / geändert / gelöscht (wer, wann, was)
  - Repository-Credentials geändert
  - Admin-Rechte vergeben / entzogen

Backup-Betrieb:
  - Backup Job gestartet / abgeschlossen / fehlgeschlagen
  - Snapshot erstellt / gelöscht

Restore:
  - Restore beantragt (wer, welcher Snapshot, wohin)
  - Restore genehmigt / abgelehnt
  - Restore abgeschlossen / fehlgeschlagen
```

```go
// Geplantes Audit-Interface
type AuditEvent struct {
    Timestamp time.Time
    Actor     string    // user-id oder agent-id
    Action    string    // "policy.created", "restore.started"
    Resource  string    // resource-id
    IP        string
    Outcome   string    // "success" | "failure"
    Details   map[string]any
}
```

---

## 8. Abhängigkeiten & Supply Chain

### Dependency-Scanning
**Status: 🔧 — Dependabot via GitHub, govulncheck fehlt in CI**

```bash
# Pflicht bei neuen Modulen
govulncheck ./...

# go mod verify prüft Checksums
go mod verify
```

### Supply-Chain-Hardening
**Status: 🔧 — teilweise**

```yaml
# CI — geplante Ergänzungen
- name: Vulnerability scan
  run: govulncheck ./...

- name: SBOM generieren
  run: syft . -o spdx-json > sbom.json

- name: Container scannen
  run: trivy image opensourcebackup:latest
```

```
✅ Dependabot konfiguriert (GitHub)
📋 govulncheck in CI
📋 go mod verify in CI
📋 GitHub Actions auf konkrete Commit-SHAs pinnen
📋 SBOM via Syft
📋 Container-Images signieren via Cosign
📋 Releases mit Checksums veröffentlichen
```

### Static Code Analysis
**Status: ✅ — golangci-lint v2 aktiv**

- `gosec` in Schicht 2 (warn) → wandert nach Schicht 1 sobald Auth implementiert
- Ziel: alle OWASP Top 10 durch Linter-Regeln abgedeckt

---

## 9. OWASP Top 10 — Mapping

| OWASP | Risiko | Status | Block |
|---|---|---|---|
| A01 Broken Access Control | Keine Auth | ❌ | B9 |
| A02 Cryptographic Failures | Kein TLS, keine Encryption at Rest | ❌ | Deployment |
| A03 Injection | SQL: ✅ parametrisiert | ✅ | — |
| A04 Insecure Design | RBAC + mTLS geplant | 📋 | B9 |
| A05 Security Misconfiguration | Security Headers fehlen | ❌ | B7 |
| A06 Vulnerable Components | govulncheck fehlt in CI | 🔧 | — |
| A07 Auth Failures | Keine Auth | ❌ | B9 |
| A08 Software Integrity | Keine Image-Scans, kein SBOM | 📋 | — |
| A09 Logging Failures | Basis ✅, Audit-Events fehlen | 🔧 | — |
| A10 SSRF | Noch kein ext. HTTP-Client | N/A | B10 |

---

## 10. Sicherheits-Checkliste vor jedem Merge

| Prüfpunkt | Tool / Methode |
|---|---|
| Keine Credentials im Code | `gosec` G101 + Privacy-Check |
| SQL nur parametrisiert | `staticcheck` + Review |
| Input validiert (Whitelist, Längen) | Review |
| Fehler nicht ungefiltert nach außen | Review |
| Body-Größen-Limit gesetzt | Review |
| Neue Abhängigkeit geprüft | `govulncheck` |
| Pfade normalisiert und geprüft | Review |
| Audit-Event für kritische Aktionen | Review |
| `//nolint` mit Begründung | Review |
| Keine Debug-Logs in Produktion | Review |

---

## 11. Abhängigkeitsgraph — Was blockiert was

```
B7 — Security Baseline Middleware
 ├── Security Headers Middleware
 ├── Request-Body-Limit (MaxBytesReader)
 ├── ReadHeaderTimeout + IdleTimeout
 └── Tests für Headers und Limits

B8 — Input Validation Layer
 ├── Whitelist-Validation (Engine, Hostname, Cron, UUID)
 ├── Pagination-Limits
 ├── saubere 400-Fehler mit Kontext
 └── Negativtests für Validierung

B9 — Auth / Agent Enrollment
 ├── erfordert: TLS (Kap. 4)
 ├── erfordert: Enrollment-Token (einmalig, TTL 10–30 Min)
 ├── erfordert: Agent-Identität + Zertifikat
 ├── erfordert: Token-Validation-Middleware
 ├── erfordert: Rate Limiting (Auth-Endpunkte)
 ├── erfordert: erste RBAC-Struktur
 └── ermöglicht: gosec Schicht 1 (lint-strategy.md)

B10 — Agent MVP
 ├── erfordert: mTLS Agent ↔ Control Plane
 ├── erfordert: Agent darf nur eigene Jobs sehen
 ├── erfordert: Job-Kommando-Allowlist
 ├── erfordert: Agent-Revocation
 └── erfordert: Audit-Events für Agent-Aktionen

B13 — Restore
 ├── erfordert: eigene Restore-Berechtigung (RBAC)
 ├── erfordert: Pfad-Validierung (Path Traversal)
 ├── erfordert: Symlink-Schutz
 ├── erfordert: Audit-Log für jeden Restore
 └── optional: 4-Augen-Freigabe

Produktion
 ├── erfordert: TLS + HTTPS ❌
 ├── erfordert: Auth (B9) ❌
 ├── erfordert: Security Headers (B7) ❌
 ├── erfordert: Rate Limiting ❌
 ├── erfordert: govulncheck in CI 🔧
 ├── erfordert: Audit-Logging 🔧
 └── erfordert: Secrets in Vault 📋
```
