# App Security — Web, Desktop, Mobile

> Sicherheitsregeln für alle App-Schichten: Web-GUI, Mobile-App, Desktop-/Tray-Client.
> Gilt sobald eine der Schichten implementiert wird — nicht erst wenn sie fertig ist.
>
> Verwandte Dokumente:
> - [CODE_SECURITY.md](CODE_SECURITY.md) — Backend, API, Auth, Secrets, Restore
> - [Lint-Strategie](../quality/lint-strategy.md) — gosec, revive für Frontend-nahen Code
> - [Architektur](../architecture/ARCHITECTURE.md) — Web-Dashboard (React/TypeScript)
> - [Developer Guide](../developer-guide/DEVELOPER_GUIDE.md) — Testing-Anforderungen

---

## Status-Legende

| Symbol | Bedeutung |
|---|---|
| ✅ | Implementiert |
| 🔧 | Teilweise / vorbereitet |
| ❌ | Fehlt — **blockiert Produktion** |
| 📋 | Geplant — noch nicht begonnen |

---

## 1. Web-App / Web-GUI

**Status: 📋 — relevant sobald Web-UI kommt (React/TypeScript laut Architektur)**

### Token-Speicherung

```
- Auth-Token niemals in localStorage speichern
  → localStorage ist über JavaScript lesbar — XSS = sofortiger Token-Diebstahl

- Für Browser-Login:
  - HttpOnly Cookie       → JavaScript kann Token nicht lesen
  - Secure Cookie         → nur über HTTPS übertragen
  - SameSite=Strict       → kein automatisches Mitsenden bei Cross-Site-Requests
```

```typescript
// ❌ Verboten
localStorage.setItem('token', jwt)

// ✅ Token kommt als HttpOnly-Cookie vom Server — kein JS-Zugriff nötig
// fetch('/api/login', { credentials: 'include' })
```

### CSRF-Schutz

```
- CSRF-Token für alle state-verändernden Requests: POST, PUT, PATCH, DELETE
- Kein CSRF-Token nötig wenn: ausschließlich Bearer-Token-Auth (kein Cookie)
- SameSite=Strict reduziert CSRF-Risiko — ersetzt aber kein explizites Token
```

### Content Security Policy

```http
Content-Security-Policy: default-src 'self';
  script-src 'self';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data:;
  connect-src 'self' https://api.opensourcebackup.internal;
  frame-ancestors 'none'
```

### Sicheres React

```typescript
// ❌ Verboten — führt XSS aus
<div dangerouslySetInnerHTML={{ __html: userInput }} />

// ✅ React escapet automatisch
<div>{userInput}</div>

// ✅ Alle API-Responses typisieren
interface System {
  id: string
  hostname: string
  riskClass: 'standard' | 'critical'
}
const data: System = await response.json()
```

### Frontend ≠ Sicherheitsgrenze

```
- Frontend-Routing darf keine Rechte ersetzen — Backend entscheidet immer
- Admin-Funktionen nie nur im UI verstecken, sondern im Backend blockieren
- Keine sensiblen Daten in:
  - Browser-Storage (localStorage, sessionStorage, IndexedDB)
  - URLs / Query-Params (landen in Logs, Browser-History, Referrer-Header)
  - console.log / Error-Messages
```

### Session-Management

```
- Session-Timeout und Logout müssen serverseitig wirksam sein
- Client-seitiger Logout allein reicht nicht — Token muss serverseitig invalidiert werden
- Inaktivitäts-Timeout: nach X Minuten automatisch ausloggen
```

---

## 2. Mobile-App

**Status: 📋 — falls Mobile-App geplant wird**

### Secrets & Token-Speicherung

```
- Secrets niemals in die App einbauen (kein hardcoded API-Key, kein Zertifikat im Bundle)
- Tokens nur im sicheren Betriebssystem-Speicher:
  - iOS:     Keychain
  - Android: Keystore / EncryptedSharedPreferences
```

```swift
// iOS — ✅ Keychain
let query: [String: Any] = [
    kSecClass as String: kSecClassGenericPassword,
    kSecAttrAccount as String: "auth_token",
    kSecValueData as String: tokenData
]
SecItemAdd(query as CFDictionary, nil)

// ❌ Verboten
UserDefaults.standard.set(token, forKey: "auth_token")
```

### Datenschutz

```
- Keine sensiblen Daten in Push Notifications (Text ist auf Lockscreen sichtbar)
- Keine sensiblen Daten in Screenshots / App-Switcher-Vorschau
  → iOS: FLAG_SECURE / ignoreScreenshot
  → Android: FLAG_SECURE für Activities mit sensiblen Inhalten
- Lokale Datenbank verschlüsseln wenn sie sensible Daten enthält (SQLCipher)
- Logout löscht lokale Tokens und sensible Caches vollständig
```

### Netzwerk

```
- TLS erzwingen — kein HTTP erlaubt
- Optional: Certificate Pinning für Agent/Admin-Kommunikation
  → Schutz gegen Man-in-the-Middle in unsicheren Netzen
  → Achtung: erhöhter Aufwand bei Zertifikats-Rotation
- ATS (App Transport Security) auf iOS nicht deaktivieren
```

### Berechtigungen & Angriffsfläche

```
- App-Berechtigungen minimal halten — nur was wirklich gebraucht wird
- Deep Links strikt validieren — kein offenes URL-Schema
- Root-/Jailbreak-Erkennung nur als Zusatzsignal, nicht als alleinige Sicherheit
  → kann umgangen werden — keine Sicherheitsgarantie
- Agent/Client darf nur erlaubte Kommandos ausführen (Allowlist)
```

### Referenz: OWASP MASVS / MASTG

Für Mobile-App-Entwicklung: [OWASP Mobile Application Security Verification Standard](https://mas.owasp.org/MASVS/)

---

## 3. Desktop-App / Tray-App

**Status: 📋 — falls Desktop-Client oder Tray-App geplant wird**

### Distribution & Updates

```
- Code Signing für alle Releases (macOS: Notarisierung, Windows: Authenticode)
- Auto-Updates nur signiert akzeptieren — ungültige Signatur = Update ablehnen
- Download-URL und Checksums in Release-Notes veröffentlichen
```

### Berechtigungen & Isolation

```
- Keine Admin-Rechte, außer wirklich nötig
- Least Privilege: App läuft als normaler User
- Sichere IPC-Kommunikation:
  - Windows: Named Pipes mit ACL
  - Linux/macOS: Unix Sockets mit Dateisystem-Berechtigungen
```

### Konfiguration & Logs

```
- Lokale Konfigurationsdateien mit restriktiven Dateirechten (0600 / Owner only)
- Keine Secrets in Klartext-Konfigurationsdateien
  → OS-Credential-Store (macOS Keychain, Windows Credential Manager, Linux libsecret)
- Lokale Logs dürfen enthalten:
  ✅ Timestamps, Hostnamen, Job-IDs, Status-Codes
  ❌ Tokens, Repository-Secrets, Passwörter, Kundendaten
```

### Auditierbare Aktionen

```
Folgende Aktionen müssen geloggt werden (lokal + an Control Plane gemeldet):
- Updates installiert
- Restore-Aktion gestartet / abgeschlossen
- Policy-Änderung empfangen und angewendet
- Verbindung zur Control Plane hergestellt / unterbrochen
- Fehler bei Job-Ausführung
```

---

## 4. App Testing

**Status: 📋 — Tests wachsen mit der jeweiligen Schicht**

### Web-Tests

```
- XSS-Tests: User-Input in allen UI-Bereichen auf Script-Injection testen
- CSRF-Tests: Requests ohne gültiges CSRF-Token müssen abgelehnt werden
- Header-Tests: alle Security Headers in jeder Response vorhanden
- Referenz: OWASP ASVS (Application Security Verification Standard)
```

### Authorization-Negativtests

```
Nicht nur testen was funktioniert — auch testen was nicht funktioniert:

- viewer  darf keine Policy ändern               → 403
- viewer  darf keine Jobs löschen                → 403
- operator darf keine Admins verwalten           → 403
- agent   darf keine fremden Jobs lesen          → 403
- agent   darf keine Jobs anderer Systeme starten → 403
- ungültiger Token                               → 401
- gültiger Token ohne Rechte                     → 403
- gelöschter Agent-Token                         → 401
- abgelaufener Token                             → 401
```

### Mobile-Tests

```
- OWASP MASTG Checkliste verwenden
- Statische Analyse: MobSF (Mobile Security Framework)
- Dynamische Analyse: Frida, objection
- Prüfen: Zertifikats-Validierung wirklich aktiv (kein Bypass möglich)
- Prüfen: Keine sensiblen Daten in Backups (Android/iOS System-Backup)
```

### Gemeinsame Regeln

```
- Keine sensiblen Daten in:
  - Browser DevTools / Network-Tab
  - App-Logs (adb logcat, Xcode Console)
  - Crash-Reports (Sentry, Firebase Crashlytics)
  - Error-Messages die dem User angezeigt werden
```

---

## 5. Abhängigkeiten zu CODE_SECURITY.md

```
CODE_SECURITY.md (Backend)          APP_SECURITY.md (Frontend/App)
─────────────────────────────       ──────────────────────────────
Auth / JWT (Kap. 1)           ←→    Token-Speicherung (Kap. 1, 2)
Security Headers (Kap. 4)     ←→    CSP / CSRF (Kap. 1)
RBAC (Kap. 1)                 ←→    Authorization-Negativtests (Kap. 4)
Audit-Logging (Kap. 7)        ←→    Auditierbare App-Aktionen (Kap. 3)
TLS (Kap. 4)                  ←→    Certificate Pinning (Kap. 2)
Agent-Sicherheit (Kap. 2)     ←→    Desktop-Agent (Kap. 3)
Restore-Sicherheit (Kap. 5)   ←→    Restore-Audit Desktop (Kap. 3)
```

---

## 6. Produktions-Checkliste App

| Schicht | Prüfpunkt | Status |
|---|---|---|
| Web | Kein Token in localStorage | 📋 |
| Web | HttpOnly + Secure + SameSite Cookies | 📋 |
| Web | CSRF-Schutz aktiv | 📋 |
| Web | CSP-Header gesetzt | ❌ B7 |
| Web | kein `dangerouslySetInnerHTML` | 📋 |
| Web | Frontend-Routing ≠ Auth | 📋 |
| Mobile | Tokens im OS-Keystore | 📋 |
| Mobile | TLS erzwungen | 📋 |
| Mobile | Keine Secrets im Bundle | 📋 |
| Desktop | Code Signing | 📋 |
| Desktop | Auto-Update signiert | 📋 |
| Desktop | Keine Secrets in Konfigdateien | 📋 |
| Alle | Authorization-Negativtests vorhanden | 📋 |
| Alle | Keine sensiblen Daten in Logs/Errors | 📋 |
