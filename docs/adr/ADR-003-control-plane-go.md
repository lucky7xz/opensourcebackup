# ADR-003: Go für Agent und Control Plane

**Status:** Akzeptiert  
**Datum:** TBD  
**Autor:** Team  
**Ticket:** OB-003  

---

## Kontext

Agent und Control Plane müssen auf Linux (amd64, arm64) und Windows (amd64) laufen.
Der Agent muss als einzelne, abhängigkeitsfreie Binary ausgeliefert werden können.
Die Control Plane muss hohe Parallelität (100+ gleichzeitige Agent-Verbindungen) effizient handhaben.

## Entscheidung

Wir verwenden **Go 1.22+** für Agent und Control Plane.
Das Web-Dashboard wird in **TypeScript 5 / React 18** implementiert.

## Begründung

| Kriterium | Go | Python | Rust | Java |
|---|---|---|---|---|
| Single Binary | ✅ | ❌ | ✅ | ❌ |
| Cross-Compile (Windows/Linux) | ✅ | — | ✅ (aufwändig) | ✅ |
| Parallelität | ✅ (goroutines) | ❌ (GIL) | ✅ | ✅ |
| Startup-Zeit | ✅ (< 100ms) | ✅ | ✅ | ❌ (JVM) |
| Operativer Overhead | ✅ (keine Runtime) | — | ✅ | ❌ |
| Restic selbst in Go | ✅ (gleiche Sprache) | — | — | — |

Go kompiliert zu einem einzelnen Binary ohne externe Laufzeitabhängigkeiten —
essentiell für den Agenten, der auf heterogenen Systemen laufen muss.

## Konsequenzen

**Positive Konsequenzen:**
- Agenten-Binary ohne Installationsaufwand verteilbar
- Restic und Borg-Wrapper in derselben Sprache wie die Engine-Autoren
- Hervorragende Stdlib für HTTP-Server, TLS, Crypto, JSON

**Negative Konsequenzen / Risiken:**
- Go's Fehlerbehandlung erfordert Disziplin (→ Clean Code Regeln)
- Kein echtes generics-basiertes ORM — SQL-First mit `pgx` / `sqlc`

**Folgeaufgaben:**
- [ ] `golangci-lint`-Konfiguration mit Pflicht-Lintern (OB-030)
- [ ] `sqlc` für typsicheres SQL evaluieren (OB-031)
- [ ] Go 1.22 Minimum in `go.mod` setzen

## Verwandte Entscheidungen

- [ADR-002](ADR-002-catalog-postgresql.md) — `pgx` als PostgreSQL-Treiber
