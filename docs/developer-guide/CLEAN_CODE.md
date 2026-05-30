# Clean Code & Wertesystem

> Verbindliche Qualitätsprinzipien für das OpensourceBackup-Projekt.
> Basiert auf dem Clean Code Developer (CCD) Wertesystem.

Dieses Dokument legt fest, nach welchen Prinzipien wir Code schreiben.
Es ist kein Vorschlag — es ist der Standard, an dem Code-Reviews gemessen werden.

---

## Die vier Grundwerte

Alles, was wir tun, dient diesen vier Werten:

| Wert | Kernthese |
|---|---|
| **Wandelbarkeit** | Software muss so gebaut sein, dass Änderungen günstig bleiben. Kosten pro Feature dürfen nicht mit der Zeit exponentiell steigen. |
| **Korrektheit** | Korrektheit wird während der Entwicklung eingebaut — nicht danach geprüft. Automatisierte Tests, nicht der Debugger, sind das Werkzeug. |
| **Produktionseffizienz** | Unnötiger Aufwand ist verschwendetes Budget. Automatisierung, klarer Code, wenig Nacharbeit. |
| **Kontinuierliche Verbesserung** | Ohne Reflexion keine Weiterentwicklung. Wir lernen täglich, als Individuen und als Team. |

---

## Prinzipien — Pflicht ab Tag 1

### DRY — Don't Repeat Yourself
**Wert: Wandelbarkeit**

Wiederhole keine Logik. Jede Information hat eine einzige, autoritative Quelle.

```go
// ❌ DRY-Verletzung
backupSizeBytes := job.BytesUploaded * 1.19
storageCostEur  := job.StorageGB * 1.19

// ✅ DRY eingehalten
func withVAT(net float64) float64 { return net * 1.19 }
backupSizeBytes := withVAT(job.BytesUploaded)
storageCostEur  := withVAT(job.StorageGB)
```

**Im Backup-Kontext:** Repository-Konfiguration, Retention-Policies, Verschlüsselungsparameter
dürfen nie an mehreren Stellen stehen. Eine Policy-Engine, eine Wahrheitsquelle.

---

### KISS — Keep It Simple
**Wert: Wandelbarkeit, Produktionseffizienz**

Die einfachste Lösung, die funktioniert, ist die richtige.
Cleverer Code ist schlechter Code.

```go
// ❌ Zu clever
result := func(x int) int { return x * (x + 1) / 2 }(len(snapshots))

// ✅ Lesbar
snapshotCount := len(snapshots)
totalPairs := snapshotCount * (snapshotCount + 1) / 2
```

**Warnsignal:** Du verstehst deinen eigenen Code nach 2 Wochen nicht mehr.

---

### IOSP — Integration Operation Segregation Principle
**Wert: Wandelbarkeit, Korrektheit**

Eine Funktion enthält entweder **Logik** oder **Aufrufe** — niemals beides.

- **Operation:** Enthält Logik (Transformationen, Bedingungen, Berechnungen)
- **Integration:** Ruft nur andere Funktionen auf, keine eigene Logik

```go
// ❌ Gemischt (IOSP-Verletzung)
func processBackupJob(job Job) error {
    if job.SizeBytes > 10*1024*1024*1024 { // Logik
        job.Priority = PriorityLow
    }
    if err := catalog.SaveJob(job); err != nil { // Integration
        return err
    }
    return notifyMonitoring(job) // Integration
}

// ✅ Getrennt
func assignPriority(job Job) Job { // Operation: nur Logik
    if job.SizeBytes > 10*GB {
        job.Priority = PriorityLow
    }
    return job
}

func processBackupJob(job Job) error { // Integration: nur Aufrufe
    job = assignPriority(job)
    if err := catalog.SaveJob(job); err != nil {
        return err
    }
    return notifyMonitoring(job)
}
```

---

### FCoI — Favour Composition over Inheritance
**Wert: Wandelbarkeit**

Interfaces und Komposition statt Vererbungshierarchien.

```go
// ✅ Komposition über Vererbung — Engine-Wrapper-Pattern
type BackupEngine interface {
    RunBackup(ctx context.Context, job Job) (Snapshot, error)
    RunRestore(ctx context.Context, snapshot Snapshot, target string) error
    ListSnapshots(ctx context.Context, repo Repository) ([]Snapshot, error)
    Prune(ctx context.Context, repo Repository, policy RetentionPolicy) error
}

// Jede Engine implementiert das Interface — keine gemeinsame Basisklasse
type ResticEngine struct { binaryPath string }
type BorgEngine   struct { binaryPath string; sshKey string }
```

---

### SRP — Single Responsibility Principle
**Wert: Wandelbarkeit**

Eine Klasse/Datei hat genau eine Verantwortlichkeit — einen Grund zur Änderung.

**Im Backup-Kontext:**
- `catalog/snapshot.go` — nur Snapshot-Datenbankoperationen
- `scheduler/job_runner.go` — nur Job-Ausführungslogik
- `api/systems.go` — nur System-API-Handler
- `agent/restic_engine.go` — nur Restic-Wrapper

**Warnsignal:** Datei über 300 Zeilen oder Methode über 30 Zeilen — Verantwortlichkeiten prüfen.

---

### SoC — Separation of Concerns
**Wert: Wandelbarkeit, Korrektheit**

Business-Logik, Infrastruktur und Präsentation sind strikt getrennt.

```
server/
├── domain/         # Business-Logik — kein HTTP, keine DB
│   ├── backup/
│   └── restore/
├── catalog/        # Nur Datenbankzugriff
├── api/            # Nur HTTP-Handler — ruft domain/ auf
└── scheduler/      # Nur Job-Scheduling
```

Domain-Code importiert niemals `net/http` oder Datenbankpakete.

---

### DIP — Dependency Inversion Principle
**Wert: Wandelbarkeit, Korrektheit**

High-Level-Module hängen von Interfaces ab, nicht von Implementierungen.

```go
// ❌ High-Level kennt Low-Level direkt
type JobRunner struct {
    catalog PostgresCatalog  // konkrete Implementierung
    engine  ResticEngine     // konkrete Implementierung
}

// ✅ Abhängigkeiten via Interfaces
type JobRunner struct {
    catalog CatalogStore   // Interface
    engine  BackupEngine   // Interface
}
// → JobRunner kann mit jedem Catalog und jeder Engine getestet werden
```

---

### Keine verfrühte Optimierung (BPO)
**Wert: Wandelbarkeit, Produktionseffizienz**

Optimiere erst nach Profiler-Messung. Verständlichkeit schlägt Performance
bis der Beweis für das Gegenteil vorliegt.

```go
// ❌ Optimierung ohne Messung
var snapshotCache sync.Map  // „könnte schneller sein"

// ✅ Erst messen, dann optimieren
// Profiler zeigt: catalog.ListSnapshots ist Bottleneck bei 10.000+ Snapshots
// → Dann und nur dann: Caching-Schicht einführen
```

---

## Praktiken — Täglich einhalten

### Boy Scout Rule (Pfadfinderregel)

Hinterlasse jeden Code in einem besseren Zustand als du ihn vorgefunden hast.
Kleine Verbesserungen, aber konsequent — kein Aufschieben.

Wenn du in einer Datei arbeitest und einen Tippfehler im Kommentar siehst:
korrigiere ihn. Wenn du einen fehlenden Test siehst: schreibe ihn.

---

### Root Cause Analysis (RCA)

Nie bei Symptomkuren stehenbleiben. Immer die Wurzel finden.

**Five Whys — Beispiel Backup-Projekt:**
```
Problem: Restore-Test schlägt fehl.
Warum?  → Snapshot ist korrupt.
Warum?  → Upload wurde bei Netzwerkunterbrechung nicht wiederholt.
Warum?  → Retry-Logik fehlt im Agent.
Warum?  → Wurde als „edge case" eingestuft und nie implementiert.
→ Lösung: Idempotentes Retry mit exponential backoff im Upload-Pfad.
```

---

### Commits atomar halten

Jeder Commit ist eine logische, abgeschlossene Einheit.
`git bisect` muss zu jedem Commit funktionieren — kein Commit darf den Build brechen.

```bash
# ❌ Zu groß
git add .
git commit -m "feat(agent): implement everything"

# ✅ Atomar
git add agent/engines/restic/runner.go agent/engines/restic/runner_test.go
git commit -m "feat(agent): add Restic backup runner with retry logic"

git add agent/engines/restic/snapshot.go agent/engines/restic/snapshot_test.go
git commit -m "feat(agent): add Restic snapshot parsing"
```

---

### Keine TODO-Kommentare ohne Ticket

```go
// ❌ Loses TODO
// TODO: handle this edge case

// ✅ Verlinktes TODO
// TODO(OB-234): handle network timeout during snapshot upload
// → Ticket OB-234 enthält Kontext, Priorität, Assignee
```

---

### Fehler explizit behandeln — nie verschlucken

```go
// ❌ Fehler verschluckt
snapshot, _ := engine.RunBackup(ctx, job)

// ❌ Fehler geloggt aber nicht propagiert
if err != nil {
    log.Error("backup failed", "err", err)
    // Caller weiß nichts davon
}

// ✅ Fehler propagiert und mit Kontext angereichert
snapshot, err := engine.RunBackup(ctx, job)
if err != nil {
    return fmt.Errorf("backup job %s: run backup: %w", job.ID, err)
}
```

---

### Keine Magie — explizit statt implizit

```go
// ❌ Magische Zahl
if job.RetryCount > 3 { ... }

// ✅ Benannte Konstante
const maxBackupRetries = 3
if job.RetryCount > maxBackupRetries { ... }
```

---

## Code-Review-Checkliste nach CCD

Folgende Fragen stellt jeder Reviewer:

| Prüfpunkt | Prinzip |
|---|---|
| Gibt es Code-Duplikation? | DRY |
| Ist die Lösung unnötig komplex? | KISS |
| Mischt die Funktion Logik und Integration? | IOSP |
| Hat die Klasse mehr als eine Verantwortlichkeit? | SRP |
| Sind Business-Logik und Infrastruktur vermischt? | SoC |
| Hängt High-Level-Code von konkreten Implementierungen ab? | DIP |
| Gibt es TODOs ohne Ticket-Referenz? | Praktik |
| Werden Fehler verschluckt oder ohne Kontext geloggt? | Praktik |
| Gibt es magische Zahlen oder Strings? | Praktik |
| Wurde ohne Profiler-Messung optimiert? | BPO |

---

## Code-Sicherheit

**Pflicht:** Jedes Feature das Auth, API, Input, Secrets, Deps, Restore oder Container berührt wird gegen
[CODE_SECURITY.md](../security/CODE_SECURITY.md) geprüft — vor dem Commit, nicht danach.

Kurzreferenz kritischer Regeln:

| Regel | Status | Block |
|---|---|---|
| Keine Credentials im Code — nur `os.Getenv` | ✅ | — |
| SQL nur parametrisiert | ✅ | — |
| Externe Fehler nicht ungefiltert nach außen | ✅ | — |
| Security Headers Middleware | ❌ | B7 |
| Request-Body-Limit + ReadHeaderTimeout | 🔧 | B7 |
| Input-Validierung (Whitelist, Längen, Pagination) | 🔧 | B8 |
| TLS / HTTPS | ❌ | Produktion |
| Auth / JWT / mTLS | ❌ | B9 |
| Agent-Enrollment (OTP-Token, Zertifikat, Revocation) | 📋 | B9 |
| Rate Limiting | ❌ | B9 |
| Pfad-Validierung bei Restore (Path Traversal) | 📋 | B13 |
| Audit-Logging (append-only) | 🔧 | B9+ |
| govulncheck in CI | 🔧 | — |
| gosec Schicht 1 | 📋 | nach B9 |

---

## Static Code Analysis — Lint-Strategie

**Wert: Korrektheit, Kontinuierliche Verbesserung**

Linting ist keine Option. Es ist das automatisierte Gedächtnis des Teams für alle
Prinzipien, die oben stehen. Was nicht automatisch geprüft wird, wird irgendwann vergessen.

### Zwei-Schichten-Modell

Wir unterscheiden **harte Regeln** (blockieren den Build) und **weiche Regeln** (Warnungen,
kein Block). Weiche Regeln werden schrittweise in harte umgewandelt — nie andersherum.

```
Schicht 1 — Hart (make lint)          → blockiert CI, kein Merge bei Fehler
Schicht 2 — Weich (make lint-warn)    → zeigt Baustellen, blockiert nie
```

**Warum nicht alles sofort hart?**
Weil ein Repo das von 0 auf 100 geht zuerst tausend Warnungen produziert, alle ignoriert
und dann niemand mehr hinschaut. Lieber 10 Regeln die wirklich gelten als 100 die niemand ernst nimmt.

---

### Schicht 1 — Harte Regeln (blockieren)

| Linter | Warum hart |
|---|---|
| `errcheck` | Verschluckte Fehler sind Bugs, keine Style-Frage |
| `govet` | Compiler-nahe Korrektheitsprüfungen |
| `staticcheck` | Tote Code-Pfade, falsche API-Nutzung |
| `unused` | Toter Code erhöht die kognitive Last dauerhaft |
| `gosimple` | Unnötige Komplexität verletzt KISS |
| `gofmt` | Formatierung ist nicht verhandelbar |
| `goimports` | Import-Ordnung ist Konvention |
| `misspell` | Tippfehler in Bezeichnern und Kommentaren |
| `bodyclose` | HTTP-Response-Bodies die nicht geschlossen werden leaken |
| `noctx` | HTTP-Requests ohne Context sind nicht produktionsreif |

---

### Schicht 2 — Weiche Regeln (Warnungen, `--exit-zero`)

Werden hart, sobald das Team sie konsistent einhält. Reihenfolge ist Priorität.

| Linter | Wert | Wann hart? |
|---|---|---|
| `revive` | Allgemeine Go-Idiome und Naming | Q3 |
| `gocritic` | Code-Qualität, Anti-Patterns | Q3 |
| `cyclop` | Zyklomatische Komplexität > 10 | Q3 |
| `funlen` | Funktionen > 60 Zeilen | Q4 |
| `godot` | Kommentare mit Punkt abschließen | Q4 |
| `exhaustive` | Nicht alle Enum-Fälle behandelt | Q4 |
| `wrapcheck` | Externe Fehler ohne `%w` gewrappt | Q4 |
| `gomnd` | Magische Zahlen (ohne Konstante) | Q4 |

---

### Workflow

```bash
# Vor jedem Commit — blockiert bei Verletzung
make lint

# Täglich oder im PR — zeigt Baustellen, blockiert nie
make lint-warn
```

**Regel:** Ein neuer Linter kommt immer zuerst in `make lint-warn`. Erst nach einem Sprint
ohne neue Verletzungen wandert er in `make lint`.

---

### Code-Review: Lint ist kein Ersatz für Review

Lint prüft **Struktur**. Review prüft **Absicht**.

```
Lint sieht:  „Diese Funktion hat 80 Zeilen."
Review fragt: „Warum hat diese Funktion 80 Zeilen — was macht sie falsch?"
```

---

## Unser Reifegrad-Ziel

Das Team arbeitet aktiv auf den **Grünen Grad** hin:

```
✅ Rot    — DRY, KISS, IOSP, FCoI, Boy Scout Rule, VCS              (Baseline: jetzt)
✅ Orange — SRP, SoC, Conventions, Reviews, Integration Tests        (Baseline: jetzt)
🔧 Gelb  — DIP, ISP, Unit Tests, Mocks, Code Coverage               (Ziel: Q3)
            → Lint-Schicht 1 (hart) + Schicht 2 (warn) vorbereitet  (✅ done)
🎯 Grün  — OCP, CI, Static Analysis scharf (alle Schicht-2 → hart)  (Ziel: Q4)
```

### Daily Reflection (für jeden Entwickler)

Am Ende jedes Arbeitstages:
- Habe ich heute DRY eingehalten?
- Habe ich Code hinterlassen, den jeder im Team in 6 Monaten versteht?
- Habe ich Fehler explizit behandelt?
- Habe ich etwas verbessert, das ich nicht selbst kaputt gemacht habe (Pfadfinderregel)?
