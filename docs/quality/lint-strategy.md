# Lint-Strategie

> Static Analysis Gate für OpensourceBackup.
> Ziel: Jede aktivierte Regel wird ernst genommen — nicht ignoriert.

---

## Grundsatz

Lint ist kein Stilwerkzeug. Es ist Teil unserer Engineering-Kultur.

Ein Lint-Fehler in `make lint` bedeutet:
- Der Code verletzt eine vereinbarte Qualitätsregel
- Der Build bleibt stehen
- Die Ursache wird behoben, nicht ignoriert
- `//nolint` ist nur mit konkreter Begründung erlaubt

Ein Lint-Hinweis in `make lint-warn` bedeutet:
- Der Code ist nicht zwingend falsch
- Es gibt eine erkennbare Qualitätsbaustelle
- Die Regel wird beobachtet
- Wenn sie stabil eingehalten wird, wandert sie in die harte Schicht

---

## Zwei-Schichten-Modell

| Schicht | Datei | Makefile | CI | Verhalten |
|---|---|---|---|---|
| Hart | `.golangci.hard.yml` | `make lint` | blockiert Merge | Exit 1 bei Verletzung |
| Weich | `.golangci.warn.yml` | `make lint-warn` | non-blocking | immer Exit 0 |

**Regel:** Neue Linter kommen immer zuerst in die weiche Schicht.
Nach einem Sprint ohne neue Verletzungen → Aufstieg in die harte Schicht.
Niemals andersherum.

---

## Schicht 1 — Harte Regeln

| Linter | Kategorie | Warum hart |
|---|---|---|
| `errcheck` | Korrektheit | Verschluckte Fehler sind Bugs |
| `govet` | Korrektheit | Compiler-nahe Prüfungen |
| `staticcheck` | Korrektheit | Tote Pfade, falsche API-Nutzung (inkl. gosimple/stylecheck) |
| `unused` | Korrektheit | Toter Code erhöht kognitive Last |
| `ineffassign` | Korrektheit | Sinnlose Zuweisungen |
| `misspell` | Lesbarkeit | Tippfehler in Bezeichnern und Kommentaren |
| `bodyclose` | Sicherheit | HTTP-Response-Bodies die nicht geschlossen werden leaken |
| `noctx` | Korrektheit | HTTP-Requests ohne Context sind nicht produktionsreif |
| `sqlclosecheck` | Korrektheit | Nicht geschlossene pgx.Rows / sql.Rows |
| `gofmt` | Formatierung | Nicht verhandelbar |
| `goimports` | Formatierung | Import-Ordnung ist Konvention |

---

## Schicht 2 — Weiche Regeln (Ziel-Quartale)

| Linter | Ziel Q | Was wird geprüft |
|---|---|---|
| `revive` | Q3 | Go-Idiome, Naming-Konventionen |
| `gocritic` | Q3 | Anti-Patterns, Code-Qualität |
| `cyclop` | Q3 | Zyklomatische Komplexität > 10 |
| `funlen` | Q4 | Funktionen > 60 Zeilen / 40 Statements |
| `godot` | Q4 | Kommentare mit Punkt abschließen |
| `exhaustive` | Q4 | Nicht alle Enum-Fälle behandelt |
| `wrapcheck` | Q4 | Externe Fehler ohne `%w` gewrappt |
| `mnd` | Q4 | Magische Zahlen ohne benannte Konstante |
| `gosec` | Q4 | Security-Schwachstellen |
| `nilnil` | Q4 | nil-Fehler und nil-Rückgabe kombiniert |
| `prealloc` | Q4 | Slice-Allokierungen ohne Kapazität |
| `rowserrcheck` | Q4 | sql.Rows.Err() nicht geprüft |

---

## Zielstufen

| Stufe | Ziel |
|---|---|
| 1 | Harte Korrektheitsregeln aktiv, Build blockiert bei echten Fehlern |
| 2 | Warn-Regeln sichtbar, aber ohne Team-Blockade |
| 3 | Komplexität und Funktionslänge werden systematisch reduziert |
| 4 | Security- und Error-Handling-Regeln werden hart |
| 5 | Lint, Tests, Migrationen und Security-Checks laufen vollständig in CI |

---

## Projektregel

Neue Features werden nur akzeptiert, wenn:

- [ ] `go test ./...` grün ist
- [ ] `make lint` grün ist
- [ ] neue Lint-Warnungen aus `make lint-warn` bewusst geprüft wurden
- [ ] keine Secrets, Tokens oder privaten Daten im Code landen
- [ ] `//nolint` nur mit konkreter Begründung verwendet wird

---

## Aufstiegsprotokoll

Wenn ein weicher Linter in die harte Schicht aufsteigen soll:

1. `make lint-warn` läuft mindestens 1 Sprint ohne neue Verletzungen für diesen Linter
2. Alle bestehenden Verletzungen werden behoben
3. Linter wird aus `.golangci.warn.yml` entfernt und in `.golangci.hard.yml` eingetragen
4. `make lint` muss danach sauber durchlaufen
5. Änderung als `chore(lint): promote <linter> to hard layer` committen

---

## nolint-Richtlinie

```go
// ❌ Verboten — keine Begründung
//nolint:errcheck

// ✅ Erlaubt — mit Begründung und Ticket
//nolint:errcheck // OB-234: pgx.Rows.Close() gibt in dieser Version immer nil zurück
```

Nolint-Kommentare ohne Begründung werden im Code-Review als Blocker behandelt.
