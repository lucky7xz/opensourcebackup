# Security Policy — OpenSourceBackup

---

## Einordnung

OpenSourceBackup implementiert **technische Bausteine zur Unterstützung eines
datenschutzgerechten Betriebs**. Das Projekt ist kein zertifiziertes Produkt
und ersetzt keine Rechtsberatung oder Sicherheitsaudit.

Für DSGVO-konformen Betrieb sind zusätzlich organisatorische Maßnahmen,
Prozessdokumentation und Rechtsgrundlagen durch den Betreiber erforderlich.

---

## Implementierte technische Maßnahmen

| Bereich | Maßnahme |
|---|---|
| **Authentifizierung** | bcrypt Admin-Passwort (Cost 12), 8h Sessions, HttpOnly-Cookies |
| **Brute-Force** | Rate-Limit: 5 Login-Versuche/min pro IP |
| **CSRF** | Double-Submit-Cookie (X-CSRF-Token), Agent-Routen ausgenommen |
| **Transport** | HTTPS/TLS konfigurierbar; Pflicht für Produktion |
| **Datenverschlüsselung** | Restic: AES-256-CTR + Poly1305 client-seitig |
| **Token-Sicherheit** | SHA-256-Hashes, nie Klartext in DB oder Logs |
| **Audit-Log** | Append-only, IP-gehasht (SHA-256), keine Secrets |
| **GDPR Art. 17** | Purge-Endpunkt für Katalogdaten |
| **GDPR Art. 20** | Export-Endpunkt für alle Metadaten |
| **Security-Header** | CSP, HSTS, X-Frame-Options, Permissions-Policy |
| **Input-Schutz** | Parametrisierte Queries, Body-Limit (1 MB), Request-Timeout |
| **Rate-Limiting** | 20 req/s global pro IP, automatisches Cleanup |

---

## Bekannte Einschränkungen (MVP)

| # | Einschränkung | Roadmap |
|---|---|---|
| 1 | `unsafe-inline` in CSP (Vite/Fonts) | Self-hosted Fonts + CSP-Nonces |
| 2 | Kein RBAC — nur globaler Admin | B_RBAC (nächste Iteration) |
| 3 | Kein MFA | Optional TOTP |
| ~~4~~ | ~~Kein DB-Level Audit-Log-Schutz~~ | ✅ Implementiert — Migration 000013, RLS |
| 5 | TLS ist opt-in, nicht Pflicht | TLS-Enforcement-Flag |
| 6 | GDPR-Purge löscht nur Katalogdaten | Dokumentierter manueller Prozess für Restic-Repo |

---

## Sicherheitslücke melden

**Bitte keine öffentlichen GitHub-Issues für Sicherheitslücken.**

Kontakt: Issues mit Label `security` oder direkt via GitHub Security Advisories.

Wir bestätigen den Eingang innerhalb von 72 Stunden.

---

## Sicherheits-Regel für dieses Projekt

> **Keine neue sicherheitsrelevante Funktion ohne:**
> - Tests (Unit und/oder Integration)
> - Audit-Log-Verhalten dokumentiert
> - Secret-Leak-Prüfung (kein Passwort/Token in Logs)
> - Threat-Model-Update (docs/security/threat-model.md)
> - Privacy-Check (speichert die Funktion personenbezogene Daten?)

---

## Weiterführende Dokumente

| Dokument | Inhalt |
|---|---|
| [docs/security/threat-model.md](docs/security/threat-model.md) | STRIDE-Analyse, offene Risiken |
| [docs/security/tom.md](docs/security/tom.md) | Technische und Organisatorische Maßnahmen (Art. 32) |
| [docs/security/gdpr-notes.md](docs/security/gdpr-notes.md) | DSGVO-Hinweise für Betreiber, Betroffenenrechte |
| [docs/security/audit-log.md](docs/security/audit-log.md) | Audit-Log-Design, IP-Hashing, Retention |
