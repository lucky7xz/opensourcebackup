# ADR-002: PostgreSQL als Katalog-Datenbank

**Status:** Akzeptiert  
**Datum:** TBD  
**Autor:** Team  
**Ticket:** OB-002  

---

## Kontext

Der zentrale Katalog speichert alle Metadaten: Systeme, Repositories, Policies,
Backup-Jobs, Snapshots, Restore-Tests, Audit-Logs. Bei 100+ Systemen mit mehrmals
täglich laufenden Backup-Jobs entstehen schnell Millionen von Datensätzen.
Anforderungen: ACID-Transaktionen, relationale Abfragen über mehrere Tabellen,
JSON-Felder für flexible Metadaten, skalierbar bis ~10 Millionen Datensätze.

## Entscheidung

Wir verwenden **PostgreSQL 16** als Katalog-Datenbank.

## Begründung

- ACID-Transaktionen: essentiell für konsistenten Job- und Snapshot-Status
- Relationale Abfragen: Jobs ↔ Snapshots ↔ Systeme ↔ Policies sind relational
- JSONB: flexible Metadaten (Engine-Output, Tags) ohne Schema-Migration
- pgBackRest-Integration: die Datenbank selbst wird via pgBackRest gesichert
- Betriebsreife: bekanntes Betriebsmodell, gute Tooling-Unterstützung

## Betrachtete Alternativen

### Option A: SQLite

Gut für MVP und Einzelinstanz. Nicht geeignet für concurrent writes bei 100+ Agenten.
**Abgelehnt** für Produktion, aber als Entwicklungs-Default akzeptabel.

### Option B: CockroachDB / TiDB

Horizontale Skalierung nicht erforderlich für geplante Größenordnung.
Operativer Overhead zu hoch für ein Open-Source-Projekt in dieser Phase.

### Option C: MongoDB / DocumentDB

Keine starken Transaktionsgarantien bei Early Versions. Relationale Abfragen
über Jobs/Snapshots/Systeme werden mit einem Dokument-Modell umständlich.

## Konsequenzen

**Positive Konsequenzen:**
- Starke Konsistenzgarantien
- Effiziente Abfragen über komplexe Join-Pfade
- JSONB für Engine-spezifische Metadaten ohne Migrations-Overhead

**Negative Konsequenzen / Risiken:**
- PostgreSQL muss selbst betrieben und gesichert werden
- Vertikale Skalierung hat Grenzen bei sehr hohem Write-Throughput

**Folgeaufgaben:**
- [ ] Migrations-Framework wählen: `golang-migrate` (OB-020)
- [ ] Initiales Schema implementieren (OB-021)
- [ ] pgBackRest-Backup der Katalog-DB konfigurieren (OB-022)
- [ ] Connection-Pool-Sizing dokumentieren

## Verwandte Entscheidungen

- [ADR-001](ADR-001-backup-engine-restic.md) — Restic Snapshot-IDs werden im Katalog gespeichert
- [ADR-003](ADR-003-control-plane-go.md) — Go-Datenbankzugriff via `pgx`
