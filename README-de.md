# GitPulseMLC

**Überwache den Herzschlag deiner lokalen Git-Repositories.**

Behalte den Überblick über alle deine Projekte – schnell, sicher und direkt im Terminal.

---

## 🇩🇪 Zusammenfassung

GitPulseMLC ist ein performantes Go-basiertes Dashboard zur Überwachung einer Vielzahl lokaler Git-Repositories. Es wurde entwickelt, um Entwicklern, die an vielen Projekten gleichzeitig arbeiten, eine schnelle Antwort auf die täglichen Fragen zu geben: Habe ich irgendwo vergessen zu pushen? Liegen noch ungespeicherte Änderungen in einem alten Branch? Sind meine lokalen Repositories auf dem Stand des Servers?

### Kern-Features
*   **Blitzschnell**: Nutzt Go-Concurrency, um hunderte Repositories in Sekunden zu scannen.
*   **Read-Only & Sicher**: Das Tool führt keine Schreibvorgänge oder automatische Merges aus. Keine Netzwerkzugriffe (außer lokal auf FETCH_HEAD).
*   **Status auf einen Blick**: Zeigt Dirty-States mit **Dateityp-Zusammenfassung** (z.B. `+3 .go`), Ahead/Behind-Counter, Stash-Informationen und das Alter der letzten Commits.
*   **Vielfältige Ausgabe**: Unterstützt das TUI-Dashboard, **JSON** Datenexport und hochkontrastreiche **HTML-Reports** (ideal für automatisierte Status-E-Mails).
*   **Aufräum-Hilfe**: Markiert "Stale Branches" (ältestes Commit > 30 Tage) automatisch.
*   **Kompakt-Modus**: Blendet auf Wunsch alle "sauberen" Repos aus, um den Fokus auf Arbeit zu lenken.
*   **Pfad-Kürzung**: Konfigurierbare Ersetzung von Pfad-Präfixen, um die Anzeige auch bei tiefen Verzeichnisstrukturen übersichtlich zu halten.
*   **Flexible Konfiguration**: Intelligente Suche nach der Konfigurationsdatei (lokal, Home-Verzeichnis oder via Flag).

---

## 📦 Nutzung als Go-Bibliothek

GitPulseMLC ist modular aufgebaut. Die Scanning-Engine kann in eigenen Go-Projekten verwendet werden:

```bash
go get github.com/hmsoft0815/GitPulseMLC
```

Detaillierte Beispiele zur Integration findest du in der [Bibliotheks-Dokumentation](docs/LIBRARY.md).

---

##  Installation & Nutzung

### Voraussetzungen
*   Go 1.25+
*   Git (installiert und konfiguriert)

### Setup
1.  Klone das Repository.
2.  Kompiliere die Binary:
    ```bash
    go build -o gitpulse ./cmd/tui
    ```
3.  Erstelle deine `config/repos.ini` (siehe `config/repos.ini.example`):
    ```ini
    [projects]
    mein-projekt = /mnt/data2tb/dev/projekt
    web-app = /mnt/data2tb/dev/anderes/repo

    [general]
    replace_path_prefix = /mnt/data2tb
    replace_with = "@"

    [settings]
    column_name_width = 30
    show_summary = true
    compact_mode = false
    ```

### Ausführen
```bash
./gitpulse          # Standardansicht
./gitpulse -v       # Detaillierter Modus: zeigt alle Branches, Sync-Status und Stashes
./gitpulse --progress # Zeigt den Scan-Fortschritt (hilfreich bei vielen Repos)
./gitpulse --compact # Zeigt nur Repositories mit Handlungsbedarf (Dirty, Ahead, etc.)
./gitpulse --html > report.html # Erzeugt einen kontrastreichen HTML-Report
./gitpulse --json > data.json   # Exportiert Statusdaten als JSON (Siehe docs/json_schema.md)
./gitpulse --config /pfad/zu/meiner.ini # Nutzt eine spezifische Konfigurationsdatei
./gitpulse --version # Zeigt Versionsinformationen
./gitpulse --help    # Zeigt alle verfügbaren Optionen
```

### Config-Suchreihenfolge
1.  `--config` Flag
2.  `GITPULSE_CONFIG` Umgebungsvariable
3.  `config/repos.ini` (aktuelles Verzeichnis)
4.  `~/.gitpulsemlc/repos.ini` (Benutzerverzeichnis)

---

## Motivation & Credits

### Warum GitPulseMLC?
Das Verwalten von 20+ aktiven Repositories auf einem lokalen Server oder einer Workstation führt oft zu "vergessenen" Commits oder veralteten lokalen Branches. Bestehende GUI-Tools sind oft zu schwerfällig. GitPulseMLC wurde entwickelt, um eine "Single Source of Truth" zu bieten, die so schnell wie `ls`, aber so informativ wie `git status` ist.

### Credits
*   **Engine**: Basiert auf der exzellenten [go-git](https://github.com/go-git/go-git) Library.
*   **UI**: Gestylt mit [Lip Gloss](https://github.com/charmbracelet/lipgloss).
*   **Inspiration**: Entwickelt als praktisches Werkzeug für hocheffiziente Entwicklungsumgebungen.

---

## Lizenz

Copyright (c) 2026 Michael Lechner.
Dieses Projekt ist unter der MIT-Lizenz lizenziert - siehe die Datei `LICENSE` für Details.
