# Changelog

Alle nennenswerten Änderungen am OpensourceBackup-Projekt werden in dieser Datei dokumentiert.

Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.1.0/).
Versionierung folgt [Semantic Versioning](https://semver.org/lang/de/).

---

## Konventionen

### Abschnitte pro Version

Jede Version kann folgende Abschnitte enthalten:

| Abschnitt | Verwendung |
|---|---|
| `### Added` | Neue Funktionalität |
| `### Changed` | Änderungen an bestehender Funktionalität |
| `### Deprecated` | Funktionalität, die in einer zukünftigen Version entfernt wird |
| `### Removed` | Entfernte Funktionalität |
| `### Fixed` | Bug-Fixes |
| `### Security` | Sicherheitsrelevante Änderungen — immer zuerst |

### Regeln

- Neueste Version steht oben
- Jeder Eintrag referenziert ein Ticket (`OB-NNN`) oder einen PR (`#NNN`)
- Breaking Changes werden mit `⚠️ BREAKING` markiert
- Jede `feat` oder `fix` Commit-Zeile landet hier — automatisch via `semantic-release`
- `[Unreleased]` enthält alle Änderungen seit dem letzten Release

---

## [Unreleased]

### Added
- Projektstruktur und initiale Dokumentation (README, Developer Guide, Clean Code, ADR-Rahmen)

---

## [0.1.0] — TBD

### Added
- Initiales Projekt-Setup
- Go-Modul-Struktur für `agent/` und `server/`
- Docker Compose Entwicklungsumgebung
- PostgreSQL-Katalog mit initialen Migrationen
- Pre-commit Hook-Konfiguration (golangci-lint, commitlint)
- CI/CD-Pipeline (GitHub Actions): lint, test, build
- Projektdokumentation: README, Developer Guide, Clean Code, CHANGELOG

---

<!-- 
VORLAGE FÜR NEUE VERSIONEN:

## [X.Y.Z] — YYYY-MM-DD

### Security
- Kurze Beschreibung ([OB-NNN], [#NNN])

### Added
- Kurze Beschreibung ([OB-NNN], [#NNN])

### Changed
- ⚠️ BREAKING: Kurze Beschreibung — Migration: [Link zum Migration Guide] ([OB-NNN])

### Fixed
- Kurze Beschreibung ([OB-NNN], [#NNN])

### Removed
- Kurze Beschreibung ([OB-NNN], [#NNN])
-->

[Unreleased]: https://github.com/your-org/opensourcebackup/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/your-org/opensourcebackup/releases/tag/v0.1.0
