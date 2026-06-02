# Developer Guide

> Verbindlicher Leitfaden für alle Entwickler am OpensourceBackup-Projekt.
> Dieses Dokument ist nicht optional — es definiert, wie wir arbeiten.

---

## Inhaltsverzeichnis

1. [Einstieg & Setup](#1-einstieg--setup)
2. [Branching & Git-Workflow](#2-branching--git-workflow)
3. [Commit-Konventionen](#3-commit-konventionen)
4. [Code-Review-Prozess](#4-code-review-prozess)
5. [Testing-Anforderungen](#5-testing-anforderungen)
6. [Definition of Done](#6-definition-of-done)
7. [Versionierung](#7-versionierung)
8. [Release-Prozess](#8-release-prozess)
9. [Sicherheitsregeln](#9-sicherheitsregeln)
10. [Kommunikation & Entscheidungen](#10-kommunikation--entscheidungen)

---

## 1. Einstieg & Setup

### Voraussetzungen

```bash
# Pflicht
go 1.22+
node 20 LTS+
docker 24+
docker compose 2.20+
git 2.40+

# Empfohlen
golangci-lint
pre-commit
pgcli
k9s (für Kubernetes-Debugging)
```

### Lokales Setup

```bash
# 1. Repository klonen
git clone https://github.com/your-org/opensourcebackup.git
cd opensourcebackup

# 2. Pre-commit Hooks installieren (PFLICHT)
pre-commit install
pre-commit install --hook-type commit-msg

# 3. Abhängigkeiten installieren
go mod download
cd web && npm ci && cd ..

# 4. Lokale Umgebung starten
docker compose -f deployments/docker-compose/dev.yml up -d

# 5. Datenbank migrieren
go run ./server/catalog/migrate up

# 6. Tests ausführen
make test
```

### Umgebungsvariablen

Alle Umgebungsvariablen werden über `.env.local` gesetzt (nie committen).
Vorlage: `.env.example` im Repository-Root — immer aktuell halten.

```bash
cp .env.example .env.local
# .env.local befüllen — niemals in VCS committen
```

### Cross-Compilation

```powershell
# Linux amd64 (Proxmox LXC / Debian)
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -ldflags="-s -w" -o osb-server-linux ./cmd/control-plane

# FreeBSD amd64 (OPNsense)
$env:GOOS="freebsd"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -ldflags="-s -w" -o opensourcebackup-agent-freebsd ./cmd/agent

# Windows amd64
$env:GOOS="windows"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -ldflags="-s -w" -o opensourcebackup-agent.exe ./cmd/agent
```

### Agent-Update-Prozess

**cerberus (Windows):**
```powershell
# 1. Neues Binary bauen (Windows oder Cross-Compile von Linux/Mac)
# 2. Alten Prozess beenden
Stop-Process -Name "opensourcebackup-agent" -Force -ErrorAction SilentlyContinue
# 3. Binary ersetzen
Copy-Item opensourcebackup-agent.exe C:\ProgramData\OpenSourceBackup\opensourcebackup-agent.exe -Force
# 4. Agent manuell oder via HKCU Run automatisch beim nächsten Login starten
Start-Process "C:\ProgramData\OpenSourceBackup\opensourcebackup-agent.exe"
```

**OPNsense (FreeBSD):**
```sh
# Via SSH oder OPNsense Console
service opensourcebackup stop
scp opensourcebackup-agent-freebsd root@192.168.0.41:/usr/local/bin/opensourcebackup-agent
chmod +x /usr/local/bin/opensourcebackup-agent
service opensourcebackup start
```

**Server (192.168.0.72, systemd):**
```bash
# Via SSH (Key: C:\Users\Admin\.ssh\id_ed25519)
systemctl stop opensourcebackup
cp osb-server-linux /opt/opensourcebackup/opensourcebackup-server
systemctl start opensourcebackup
```

### Agent-Token ohne Web-UI generieren

Falls CSRF oder UI-Problem den normalen Enrollment-Flow blockiert, Token direkt per DB + Python:

```python
import hashlib, secrets, base64, psycopg2

# Plain-Token generieren (32 random bytes, base64url)
plain = base64.urlsafe_b64encode(secrets.token_bytes(32)).rstrip(b'=').decode()
token_hash = hashlib.sha256(plain.encode()).hexdigest()

conn = psycopg2.connect("host=localhost dbname=opensourcebackup user=postgres")
cur = conn.cursor()
cur.execute(
    "INSERT INTO agent_tokens (system_id, token_hash) VALUES (%s, %s)",
    ('<system-uuid>', token_hash)
)
conn.commit()
print("Plain token (in agent-token file):", plain)
```

Den `plain`-Wert in die Token-Datei des Agents schreiben (kein Zeilenumbruch, Rechte 0600 / auf Windows nur für den jeweiligen User).

---

## 2. Branching & Git-Workflow

Wir verwenden **GitHub Flow** mit einem einzigen langlebigen Branch.

```
main (always deployable)
 ├── feature/OB-123-agent-restic-wrapper
 ├── fix/OB-456-catalog-deadlock
 ├── chore/OB-789-update-go-deps
 └── docs/OB-101-architecture-adr-001
```

### Branch-Regeln

| Regel | Details |
|---|---|
| `main` ist immer deployable | Kein direktes Pushen auf `main` |
| Branch-Namen | `<typ>/OB-<ticket>-<kurzbeschreibung>` |
| Kurzlebig | Feature-Branches max. 5 Werktage offen |
| Aktuell halten | Täglich `git rebase main` auf laufenden Branches |
| Löschen nach Merge | Branch wird nach dem Merge automatisch gelöscht |

### Branch-Typen

| Typ | Verwendung |
|---|---|
| `feature/` | Neue Funktionalität |
| `fix/` | Bug-Fixes |
| `chore/` | Technische Aufgaben ohne funktionale Änderung |
| `docs/` | Nur Dokumentation |
| `refactor/` | Code-Umstrukturierung ohne Verhaltensänderung |
| `test/` | Tests hinzufügen oder korrigieren |
| `ci/` | CI/CD-Pipeline-Änderungen |

---

## 3. Commit-Konventionen

Wir folgen **Conventional Commits 1.0.0**.
Commits werden automatisch für den Changelog ausgewertet.

### Format

```
<typ>(<scope>): <kurzbeschreibung>

[optionaler body]

[optionaler footer: BREAKING CHANGE, Closes #123]
```

### Typen

| Typ | Auswirkung auf Version | Verwendung |
|---|---|---|
| `feat` | Minor | Neue Funktionalität |
| `fix` | Patch | Bug-Fix |
| `docs` | — | Nur Dokumentation |
| `refactor` | — | Code-Umbau ohne Verhaltensänderung |
| `test` | — | Tests hinzufügen / korrigieren |
| `chore` | — | Build, Abhängigkeiten, CI |
| `ci` | — | CI/CD-Änderungen |
| `perf` | Patch | Performance-Verbesserung |
| `revert` | Patch | Revert eines früheren Commits |
| `BREAKING CHANGE` | **Major** | Im Footer oder `!` nach Typ |

### Scopes (Pflicht wenn zutreffend)

```
agent       server      web         catalog
scheduler   api         auth        policy
storage     monitoring  deploy      docs
```

### Beispiele

```bash
# Neue Funktion
feat(agent): add Restic engine wrapper with S3 backend support

# Bug-Fix
fix(catalog): resolve deadlock in concurrent snapshot writes

# Breaking Change
feat(api)!: rename /v1/systems to /v1/hosts

BREAKING CHANGE: All API clients must update endpoint URLs.
Closes #OB-234

# Chore
chore(deps): update Go to 1.22.3

# Docs
docs(adr): add ADR-002 for PostgreSQL catalog decision
```

### Verboten

```bash
# ❌ Keine sprechende Beschreibung
git commit -m "fix stuff"
git commit -m "wip"
git commit -m "changes"
git commit -m "."

# ❌ Kein Typ
git commit -m "added new feature for restic"

# ❌ Großbuchstabe am Anfang der Beschreibung
git commit -m "feat(agent): Add new feature"  # 'Add' → 'add'

# ❌ Punkt am Ende
git commit -m "feat(agent): add restic wrapper."
```

---

## 4. Code-Review-Prozess

### Pflichtregeln

- Jeder Pull Request braucht **mindestens 1 Approval** vor dem Merge
- PRs mit `BREAKING CHANGE` brauchen **2 Approvals**
- Der Autor des PRs darf nicht selbst mergen
- CI muss vollständig grün sein vor dem Merge
- PRs über 400 geänderte Zeilen werden aufgeteilt

### PR-Beschreibung (Template)

```markdown
## Was ändert sich?
<!-- Kurze Beschreibung der Änderung -->

## Warum?
<!-- Kontext, Ticket-Referenz: Closes #OB-xxx -->

## Wie wurde es getestet?
<!-- Unit-Tests / Integration-Tests / manuell — was genau? -->

## Checkliste
- [ ] Tests geschrieben oder aktualisiert
- [ ] CHANGELOG.md aktualisiert (bei feat/fix)
- [ ] Dokumentation aktualisiert (falls nötig)
- [ ] Kein TODO-Kommentar ohne Ticket-Referenz
- [ ] Keine neuen Sicherheitslücken eingebracht (siehe Sicherheitsregeln)
```

### Reviewer-Verhalten

**Du bist Reviewer — deine Pflichten:**
- Review innerhalb von **1 Werktag** (nicht Woche)
- Konkrete Verbesserungsvorschläge, kein „das gefällt mir nicht"
- Schweregrad angeben: `nit:` (optional), `suggestion:` (empfohlen), `blocker:` (muss behoben werden)
- Lob ist erlaubt und erwünscht

**Beispiele:**

```
nit: Variablenname könnte klarer sein — vielleicht `snapshotID` statt `sid`?

suggestion: Hier könnte ein früher Return die Verschachtelung reduzieren.

blocker: Diese Funktion enthält keine Fehlerbehandlung für den Datenbankfehler.
         Wenn die DB nicht erreichbar ist, wird der Job lautlos fehlschlagen.
```

---

## 5. Testing-Anforderungen

### Pflichtabdeckung

| Ebene | Minimum-Coverage | Tool |
|---|---|---|
| Unit Tests (Go) | 80% pro Package | `go test` + `go tool cover` |
| Unit Tests (TS) | 80% pro Modul | Vitest |
| Integration Tests | Alle API-Endpunkte | `testcontainers-go` |
| Restore-Tests | Jeder neue Backup-Pfad | Eigener Testrahmen |

### Regeln

- **Kein Merge ohne Tests** für neue Funktionalität
- Tests müssen im selben PR wie der Code kommen — kein „teste ich später"
- Kein Mocking von internen Paketen — nur externe Abhängigkeiten mocken
- Test-Namen beschreiben das erwartete Verhalten, nicht die Implementierung

```go
// ❌ Schlechter Testname
func TestResticEngine(t *testing.T) { ... }

// ✅ Guter Testname
func TestResticEngine_RunBackup_ReturnsSnapshotID_WhenRepositoryExists(t *testing.T) { ... }
func TestResticEngine_RunBackup_ReturnsError_WhenRepositoryUnreachable(t *testing.T) { ... }
```

### Test-Kategorien (Build-Tags)

```go
//go:build unit
//go:build integration
//go:build e2e
//go:build restore
```

```bash
# Nur Unit-Tests (schnell, kein Docker)
make test-unit

# Integration-Tests (braucht Docker)
make test-integration

# Alle Tests
make test-all
```

---

## 6. Definition of Done

Ein Task ist **Done** wenn alle folgenden Punkte erfüllt sind:

### Code
- [ ] Code kompiliert ohne Fehler und Warnungen
- [ ] Alle Linter-Checks bestanden (`make lint`)
- [ ] Unit-Tests geschrieben, alle grün
- [ ] Integration-Tests (falls neue API-Endpunkte) geschrieben und grün
- [ ] Code-Coverage nicht gesunken

### Review
- [ ] PR erstellt mit ausgefüllter Beschreibung
- [ ] Mindestens 1 Approval erhalten
- [ ] Alle Blocker-Kommentare adressiert
- [ ] CI vollständig grün

### Dokumentation
- [ ] `CHANGELOG.md` aktualisiert (bei `feat` und `fix`)
- [ ] Inline-Dokumentation (GoDoc / TSDoc) für öffentliche Funktionen
- [ ] ADR erstellt wenn eine Architekturentscheidung getroffen wurde
- [ ] `.env.example` aktualisiert wenn neue Umgebungsvariablen hinzukamen

### Sicherheit
- [ ] Keine Secrets oder Credentials im Code
- [ ] Keine neuen Abhängigkeiten ohne Security-Prüfung (`nancy` / `npm audit`)
- [ ] Input-Validierung für alle API-Eingaben vorhanden

---

## 7. Versionierung

Wir folgen **Semantic Versioning 2.0.0** (semver.org).

```
MAJOR.MINOR.PATCH

1.0.0
│ │ └── Patch: Rückwärtskompatible Bug-Fixes
│ └──── Minor: Rückwärtskompatible neue Funktionalität
└────── Major: Breaking Changes
```

### Versions-Mapping zu Conventional Commits

| Commit-Typ | Version-Bump |
|---|---|
| `fix`, `perf`, `revert` | Patch |
| `feat` | Minor |
| `BREAKING CHANGE` / `!` | Major |
| `docs`, `chore`, `ci`, `test`, `refactor` | kein Bump |

### Pre-Release

```
1.0.0-alpha.1   # Alpha — nicht produktionsreif
1.0.0-beta.1    # Beta — Feature-complete, aber nicht stabil
1.0.0-rc.1      # Release Candidate — Produktionstest
```

---

## 8. Release-Prozess

### Automatisiert (CI/CD)

```
1. Merge auf main
      ↓
2. CI: Tests, Linter, Security-Scan
      ↓
3. semantic-release: Version berechnen, CHANGELOG generieren
      ↓
4. Git-Tag erstellen (z.B. v1.2.0)
      ↓
5. Docker-Images bauen und pushen (ghcr.io)
      ↓
6. GitHub Release mit Release Notes erstellen
      ↓
7. Helm Chart Version bumpen
```

### Manueller Hotfix

```bash
# Von main branchen
git checkout -b fix/OB-999-kritischer-fehler main

# Fix implementieren und committen
git commit -m "fix(catalog): prevent data loss on concurrent writes"

# PR → main (expedited review: 1 Approval reicht)
# Nach Merge: Release wird automatisch erstellt
```

### Release-Kommunikation

Nach jedem Release:
- GitHub Release Notes veröffentlichen (automatisch aus Changelog)
- Bei Breaking Changes: Migration Guide in `docs/migrations/` erstellen
- Bei Security-Fixes: CVE-Nummer im Release vermerken

---

## 9. Sicherheitsregeln

Diese Regeln sind nicht verhandelbar.

### Absolut verboten

```bash
# ❌ Credentials im Code
password := "supersecret123"
apiKey := "sk-prod-abc123"

# ❌ Credentials in Commit-Messages oder PR-Beschreibungen

# ❌ Selbstgebaute Kryptografie
# → immer Go's crypto/aes, crypto/tls, golang.org/x/crypto verwenden

# ❌ TLS < 1.2 konfigurieren

# ❌ SQL-Queries durch String-Konkatenation bauen
query := "SELECT * FROM users WHERE id = " + userID  // SQL-Injection
```

### Pflicht

```go
// ✅ Parametrisierte Queries immer
db.QueryRow("SELECT * FROM snapshots WHERE id = $1", snapshotID)

// ✅ Input-Validierung an API-Grenzen
// ✅ Secrets aus Vault oder Umgebungsvariablen — nie hardcoded
// ✅ mTLS zwischen Agent und Control Plane
// ✅ Minimale Berechtigungen (least privilege)
```

### Abhängigkeiten

```bash
# Bei jedem neuen Go-Modul: Security-Scan pflicht
nancy sleuth < go.sum

# Bei jedem neuen npm-Paket
npm audit

# Abhängigkeiten wöchentlich prüfen (automatisch per Dependabot)
```

### Vulnerability Reporting

Sicherheitslücken **nicht** als öffentliches GitHub Issue melden.
→ E-Mail an: security@your-org.com
→ PGP-Key: [Link zum Public Key]
→ Wir antworten innerhalb von 48 Stunden

---

## 10. Kommunikation & Entscheidungen

### Architekturentscheidungen (ADR)

Jede signifikante Architekturentscheidung wird als **ADR** dokumentiert.

**Was ist eine signifikante Entscheidung?**
- Neue Abhängigkeit (Library, Datenbank, Service)
- Änderung am API-Vertrag
- Änderung an Datenbankschema-Kernstruktur
- Wahl einer Backup-Engine oder eines Storage-Backends
- Änderung an Security-Mechanismen

```bash
# Neues ADR erstellen
cp docs/adr/ADR-000-template.md docs/adr/ADR-005-beschreibung.md
```

Format: `docs/adr/ADR-NNN-kurztitel.md` — siehe [ADR-Template](../adr/ADR-000-template.md).

### Technische Diskussionen

- GitHub Issues für Bugs und Features
- GitHub Discussions für offene Fragen und RFCs
- Keine Architekturdiskussionen in PR-Kommentaren — Issue erstellen, verlinken

### Uneinigkeit

1. Argumente schriftlich im Issue/Discussion sammeln
2. Betroffene Personen einbeziehen
3. Zeitlimit: 3 Werktage für Konsens
4. Falls kein Konsens: Projektleitung entscheidet, Entscheidung wird als ADR dokumentiert
