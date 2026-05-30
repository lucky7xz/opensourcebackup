# Code Security

> Verbindliche Sicherheitsregeln für OpensourceBackup.
> Wird bei jedem Feature das Auth, API, Input, Secrets, Deps oder Container berührt geprüft.
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
| 📋 | Geplant — kommt in B9–B13 |

---

## 1. Authentifizierung & Autorisierung

### Passwort-Hashing
**Status: 📋 (B9 — Agent Auth)**

- Niemals Klartext-Passwörter speichern
- Hashing-Algorithmus: **bcrypt** (cost ≥ 12) oder **Argon2id**
- MD5, SHA-1, SHA-256 für Passwörter: **verboten**

```go
// ✅ bcrypt
hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)

// ❌ Verboten
hash := sha256.Sum256([]byte(password))
```

### Token-basierte Authentifizierung
**Status: ❌ — blockiert Produktion**

- JWT für Benutzer-Sessions: kurzlebig (15–60 Min), signiert mit RS256 oder HS256
- mTLS für Agent ↔ Control Plane: TLS-Client-Zertifikate, keine Passwörter
- HttpOnly-Cookies für Web-Clients: verhindert XSS-Token-Diebstahl
- Refresh-Token: langlebig, rotierend, in DB invalidierbar

```go
// Token-Lebensdauer — keine magischen Zahlen
const (
    accessTokenTTL  = 15 * time.Minute
    refreshTokenTTL = 7 * 24 * time.Hour
)
```

### Session-Management
**Status: ❌ — blockiert Produktion**

- Sessions bei Logout invalidieren (Token-Blacklist oder DB-Revocation)
- SameSite=Strict für Cookies
- Keine Session-IDs in URLs

### RBAC — Role-Based Access Control
**Status: 📋 (B9)**

- Restore-Rechte getrennt von Backup-Rechten (laut Architektur)
- Rollen: `admin`, `operator`, `viewer`, `agent`
- Least Privilege: jeder Principal bekommt nur was er braucht

### MFA
**Status: 📋 — optional für v1, empfohlen für Admin-Accounts**

- TOTP (Time-based One-Time Password) via Authenticator-App
- Backup-Codes bei Einrichtung generieren und sicher speichern

---

## 2. Datenvalidierung & Bereinigung

### Whitelist-Validation statt Blacklist
**Status: 🔧 — Pflichtfelder geprüft, Formate noch nicht**

Jede API-Eingabe wird im Backend validiert — client-seitige Validierung ist nur UX, keine Sicherheit.

```go
// ✅ Whitelist: erlaubte Werte explizit definieren
var validEngines = map[string]bool{"restic": true, "borg": true, "pgbackrest": true, "velero": true}
if !validEngines[p.Engine] {
    writeError(w, http.StatusBadRequest, "unsupported engine")
    return
}

// ✅ Längen-Limits
if len(s.Hostname) == 0 || len(s.Hostname) > 253 {
    writeError(w, http.StatusBadRequest, "hostname: 1–253 chars required")
    return
}

// ✅ Cron-Ausdruck validieren bevor Speicherung
if _, err := cron.ParseStandard(schedule); err != nil {
    writeError(w, http.StatusBadRequest, "invalid cron expression")
    return
}
```

### SQL-Injection-Schutz
**Status: ✅ — vollständig**

Alle Datenbankabfragen verwenden parametrisierte Queries via pgx:

```go
// ✅ Immer so — kein String-Concat mit User-Input
pool.QueryRow(ctx, "SELECT * FROM systems WHERE id = $1", id)

// ❌ Niemals so
pool.QueryRow(ctx, "SELECT * FROM systems WHERE id = '" + id + "'")
```

**Automatisch geprüft durch:** `staticcheck` (Schicht 1)

### Script-Injection / XSS
**Status: 📋 — relevant wenn Web-UI kommt**

- Alle User-Daten in HTML-Responses escapen
- Content Security Policy (CSP) Header setzen
- React/Vue escapen automatisch — kein `dangerouslySetInnerHTML`

---

## 3. API-Sicherheit & Netzwerk

### HTTPS / TLS
**Status: ❌ — blockiert Produktion**

- TLS 1.3 minimum, TLS 1.2 als Fallback
- Kein selbstsigniertes Zertifikat in Produktion
- Let's Encrypt oder eigene CA für interne Deployments
- mTLS zwischen Agent und Control Plane (laut Architektur)

```go
// Produktion: TLS-Config
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS13,
}
```

### Rate Limiting
**Status: ❌ — fehlt**

- Alle API-Endpunkte: max. 100 Requests/Minute pro IP
- Login/Auth-Endpunkte: max. 10 Requests/Minute pro IP (Brute-Force-Schutz)
- Implementierung: Token-Bucket-Algorithmus oder `golang.org/x/time/rate`

```go
// Geplante Middleware
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler
```

### CORS
**Status: 📋 — relevant wenn Web-UI kommt**

- Strikte Origin-Whitelist — kein `Access-Control-Allow-Origin: *` in Produktion
- Nur erlaubte HTTP-Methoden und Header explizit listen

### Security Headers
**Status: ❌ — fehlt**

Folgende Header bei jeder Response setzen:

```go
// Geplante Security-Header-Middleware
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
w.Header().Set("Content-Security-Policy", "default-src 'self'")
```

### CSRF-Schutz
**Status: 📋 — relevant wenn Web-UI mit Cookies kommt**

- Anti-CSRF-Token für state-verändernde Requests (POST, PUT, DELETE)
- SameSite=Strict für Auth-Cookies

---

## 4. Sensible Daten & Umgebungsvariablen

### Secrets nie im Code
**Status: ✅ — eingehalten**

```bash
# ✅ Aus Umgebungsvariable lesen
DATABASE_URL=postgres://user:pass@host/db

# ❌ Niemals im Code
const dbPass = "supersecret"
```

Automatisch geprüft durch:
- `.gitignore` schützt `.env.local`
- Pre-Push Privacy-Check (Memory: `feedback_pre_push_privacy_check`)
- `gosec` (G101: hardcoded credentials) — in Schicht 2, wandert zu Schicht 1

### Secrets Manager für Produktion
**Status: 📋 (Deployment)**

- HashiCorp Vault oder SOPS (laut Architektur)
- Repository-Credentials: pro System/Mandant getrennt
- Rotation: Secrets haben ein TTL, werden automatisch rotiert

### Verschlüsselung at Rest
**Status: 📋 (PostgreSQL-Deployment)**

- PostgreSQL-Daten: Disk-Verschlüsselung (LUKS / cloud-native)
- Backup-Daten: Verschlüsselung beim Agenten — kein Vertrauen in Storage
- Encryption Keys: niemals zusammen mit verschlüsselten Daten speichern

### Container-Sicherheit
**Status: 📋 (Dockerfile kommt)**

```dockerfile
# ✅ Minimales Base-Image
FROM gcr.io/distroless/static-debian12

# ✅ Non-root User
USER nonroot:nonroot

# ❌ Verboten
FROM ubuntu:latest
RUN apt-get install -y ...
```

- `latest`-Tags verboten — immer Pin auf konkrete Version
- Image-Scans via Trivy oder Grype in CI

---

## 5. Fehlerbehandlung & Logging

### Generische Fehlermeldungen nach außen
**Status: ✅ — eingehalten**

```go
// ✅ Nach außen: generisch
writeError(w, http.StatusInternalServerError, "internal error")

// ✅ Intern: vollständiger Kontext
h.log.Error("database query failed", "error", err, "query", "GetByID", "id", id)

// ❌ Niemals Stack-Traces oder DB-Details nach außen
writeError(w, 500, err.Error()) // kann interne Struktur leaken
```

### Security-Events loggen
**Status: 🔧 — Basis-Logging vorhanden, Security-Events fehlen**

Folgende Events müssen explizit geloggt werden:
- Fehlgeschlagene Authentifizierungsversuche (mit IP)
- Unautorisierte Zugriffe (403)
- Ungewöhnliche Request-Patterns
- Änderungen an Policies oder Retention-Regeln

---

## 6. Abhängigkeiten & DevSecOps

### Dependency-Scanning
**Status: 🔧 — Dependabot via GitHub vorbereitet, govulncheck fehlt**

```bash
# Bei jedem neuen Go-Modul pflicht
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Automatisch via GitHub Dependabot (konfiguriert)
```

Geplante CI-Ergänzung:
```yaml
- name: Vulnerability scan
  run: govulncheck ./...
```

### CI/CD Security Gates
**Status: 🔧 — Lint und Tests laufen, Security-Scan fehlt**

Vollständige Pipeline (Ziel):
```
Push → Lint (hard) → Tests → govulncheck → gosec → Image-Scan → Deploy
```

Aktuell:
```
Push → Lint (hard) → Tests ✅
```

### Static Code Analysis
**Status: ✅ — golangci-lint v2 aktiv**

- `gosec` läuft in Schicht 2 (warn) — wandert nach Schicht 1 wenn Auth implementiert
- Ziel: alle OWASP Top 10 durch Linter-Regeln abdecken

---

## 7. Sicherheits-Checkliste vor jedem Merge

| Prüfpunkt | Tool / Methode |
|---|---|
| Keine Credentials im Code | `gosec` G101 + Privacy-Check |
| SQL nur parametrisiert | `staticcheck` + Code-Review |
| Input-Validierung vorhanden | Code-Review |
| Fehler nicht nach außen gelenkt | Code-Review |
| Neue Abhängigkeit geprüft | `govulncheck` + `go mod tidy` |
| `//nolint` mit Begründung | Code-Review |
| Keine Debug-Logs in Produktion | Code-Review |
| Security-Events geloggt | Code-Review |

---

## 8. OWASP Top 10 — Mapping

| OWASP | Risiko | Status |
|---|---|---|
| A01 Broken Access Control | Keine Auth | ❌ B9 |
| A02 Cryptographic Failures | Kein TLS, keine Encryption at Rest | ❌ Deployment |
| A03 Injection | SQL: ✅ parametrisiert | ✅ |
| A04 Insecure Design | RBAC geplant, mTLS geplant | 📋 B9 |
| A05 Security Misconfiguration | Security Headers fehlen | ❌ |
| A06 Vulnerable Components | govulncheck fehlt in CI | 🔧 |
| A07 Auth Failures | Keine Auth | ❌ B9 |
| A08 Software Integrity | Keine Image-Scans | 📋 |
| A09 Logging Failures | Basis-Logging ✅, Security-Events fehlen | 🔧 |
| A10 SSRF | Noch kein externer HTTP-Client | N/A |

---

## Abhängigkeiten zwischen Sicherheitsbereichen

```
B9 — Auth / Enrollment Token
 ├── erfordert: TLS (3.)
 ├── erfordert: Token-Validierung Middleware (1.)
 ├── erfordert: Rate Limiting (3.)
 └── ermöglicht: gosec Schicht 1 (lint-strategy.md)

B10 — Agent MVP
 ├── erfordert: mTLS (3.)
 └── erfordert: Least-Privilege Agent-Rolle (1. RBAC)

Produktion
 ├── erfordert: TLS + HTTPS
 ├── erfordert: Auth (B9)
 ├── erfordert: Security Headers
 ├── erfordert: Rate Limiting
 └── erfordert: govulncheck in CI
```
