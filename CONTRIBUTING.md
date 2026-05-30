# Contributing to OpensourceBackup

Danke für dein Interesse am Projekt. Dieses Dokument erklärt, wie du beitragen kannst.

---

## Bevor du anfängst

1. Lies den [Developer Guide](docs/developer-guide/DEVELOPER_GUIDE.md) vollständig
2. Lies [Clean Code & Wertesystem](docs/developer-guide/CLEAN_CODE.md)
3. Schau in die offenen [Issues](https://github.com/your-org/opensourcebackup/issues)
4. Für große Änderungen: erst ein Issue erstellen und Konzept besprechen

---

## Arten von Beiträgen

| Art | Wie |
|---|---|
| Bug gefunden | Issue erstellen mit Reproduktionsschritten |
| Feature-Idee | Issue erstellen, Discussion starten |
| Dokumentation | PR direkt — kein Issue nötig |
| Bug-Fix | Fork → Branch → PR |
| Neue Funktion | Erst Issue, dann PR nach Diskussion |
| Sicherheitslücke | **Nicht** als Issue — security@your-org.com |

---

## Der Beitragsprozess

```
1. Fork erstellen
2. Feature-Branch anlegen: git checkout -b feature/OB-NNN-kurzbeschreibung
3. Änderungen entwickeln (Developer Guide beachten)
4. Tests schreiben
5. make test && make lint — muss grün sein
6. Commits nach Conventional Commits
7. Pull Request öffnen (Template ausfüllen)
8. Code Review — Feedback adressieren
9. Merge
```

---

## Was wir nicht annehmen

- Code ohne Tests
- PRs, die die Linter-Checks nicht bestehen
- Abhängigkeiten ohne Begründung
- Breaking Changes ohne vorherige Diskussion
- Commits ohne Conventional-Commit-Format

---

## Verhaltenskodex

Wir folgen dem [Contributor Covenant](https://www.contributor-covenant.org/de/).

Kurz: Respektvoller Umgang. Sachliche Diskussionen. Kein Raum für Diskriminierung.
Verstöße melden: conduct@your-org.com

---

## Fragen?

GitHub Discussions: [Link] — wir antworten in der Regel innerhalb eines Werktags.
