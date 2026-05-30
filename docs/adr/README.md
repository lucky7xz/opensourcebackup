# Architecture Decision Records

Architecture Decision Records (ADRs) dokumentieren signifikante Architekturentscheidungen —
was wurde entschieden, warum, und welche Alternativen wurden verworfen.

---

## Index

| Nr. | Titel | Status | Datum |
|---|---|---|---|
| [ADR-000](ADR-000-template.md) | Template | Vorlage | — |
| [ADR-001](ADR-001-backup-engine-restic.md) | Restic als Standard Backup-Engine | Akzeptiert | TBD |
| [ADR-002](ADR-002-catalog-postgresql.md) | PostgreSQL als Katalog-Datenbank | Akzeptiert | TBD |
| [ADR-003](ADR-003-control-plane-go.md) | Go für Agent und Control Plane | Akzeptiert | TBD |

## Status-Werte

| Status | Bedeutung |
|---|---|
| `Vorschlag` | In Diskussion, noch nicht entschieden |
| `Akzeptiert` | Entschieden und gilt |
| `Abgelehnt` | Diskutiert, aber verworfen (Begründung im ADR) |
| `Ersetzt` | Durch neueres ADR ersetzt — Link zum Nachfolger |
| `Veraltet` | Nicht mehr relevant |

## Wann ein ADR erstellen?

- Neue externe Abhängigkeit (Library, Datenbank, Service)
- Änderung am API-Vertrag oder Datenbankschema-Kern
- Wahl einer Backup-Engine oder Storage-Backend
- Änderung an Authentifizierungs- oder Verschlüsselungsmechanismen
- Jede Entscheidung, bei der das Team 30 Minuten diskutiert hat

## Vorlage verwenden

```bash
cp docs/adr/ADR-000-template.md docs/adr/ADR-NNN-kurztitel.md
```
