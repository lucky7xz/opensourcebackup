# ADR-001: Restic als Standard Backup-Engine

**Status:** Akzeptiert  
**Datum:** TBD  
**Autor:** Team  
**Ticket:** OB-001  

---

## Kontext

Das Projekt benötigt eine oder mehrere Backup-Engines für Datei-Backups auf Linux- und
Windows-Systemen. Die Engine muss Deduplizierung, Verschlüsselung und verschiedene Storage-
Backends (lokal, S3, MinIO) unterstützen. Eine Eigenentwicklung des Repository-Formats
und der Kryptografie wurde als zu riskant eingestuft (→ Grundprinzip: keine selbstgebaute
Kryptografie).

## Entscheidung

Wir verwenden **Restic** als Standard Backup-Engine für Datei-Backups.
Borg wird als Alternative für Linux/SSH-Umgebungen unterstützt.
Die eigene Plattform orchestriert und wrapt die Engines — sie ersetzt sie nicht.

## Begründung

| Kriterium | Restic | Borg | Selbstentwicklung |
|---|---|---|---|
| Verschlüsselung (AES-256) | ✅ | ✅ | ❌ riskant |
| S3 / MinIO nativ | ✅ | ❌ (via SFTP-Wrapper) | — |
| Windows-Support | ✅ | ❌ (kein nativer Client) | — |
| Single Binary | ✅ | ❌ | — |
| Aktive Community | ✅ | ✅ | — |
| JSON-Output | ✅ (vollständig) | ✅ (teilweise) | — |
| Repository-Lock | ✅ | ✅ | — |

Restic bietet nativen S3-Support (essentiell für MinIO-Backend), Windows-Support und
vollständig maschinenlesbaren JSON-Output — wichtig für die Katalog-Integration.

## Betrachtete Alternativen

### Option A: Nur Borg

**Vorteile:** Stärkere globale Deduplizierung, sehr gute Linux/SSH-Performance.

**Nachteile:** Kein nativer Windows-Client, kein nativer S3-Support,
kein vollständiges JSON-Output-Format.

### Option B: Eigenes Repository-Format

**Vorteile:** Volle Kontrolle, keine externe Abhängigkeit.

**Nachteile:** Kryptografie, Chunking, Locking und Pruning selbst implementieren.
Das sind genau die Teile, die man in einem Backup-System nicht leichtfertig neu baut.
Einmal fehlerhaft, verliert man Daten oder hat unbemerkt unverschlüsselte Backups.
**Abgelehnt — zu hohes Risiko.**

### Option C: Restic + Borg (beide)

**Entscheidung:** Restic als Standard, Borg als unterstützte Alternative.
Beide implementieren das `BackupEngine`-Interface.

## Konsequenzen

**Positive Konsequenzen:**
- Bewährte Kryptografie (AES-256-CTR + Poly1305-AES)
- Aktive Community und regelmäßige Security-Fixes
- Voller S3/MinIO-Support ohne Wrapper
- Windows- und Linux-Support mit einer Engine

**Negative Konsequenzen / Risiken:**
- Wir sind von Restic's Release-Zyklus abhängig
- Restic-Repository-Format-Änderungen könnten Migration erfordern
- Keine Kontrolle über Repository-Lock-Verhalten bei abgestürzten Jobs

**Folgeaufgaben:**
- [ ] `agent/engines/restic/` implementieren (OB-010)
- [ ] Restic-Version pinnen und automatisch via Dependabot aktualisieren
- [ ] Restore-Test-Pfad für Restic implementieren (OB-011)

## Verwandte Entscheidungen

- [ADR-002](ADR-002-catalog-postgresql.md) — Katalog speichert Restic Snapshot-IDs
