# Alle Git-Repositories auf einen Blick

**Der Terminal-Herzschlag für Entwickler**
GitPulseMLC ist der ultimative Begleiter für Entwickler, die dutzende lokale Repositories verwalten. Anstatt jeden Ordner manuell zu prüfen, erhalten Sie in Sekundenbruchteilen einen Gesamtüberblick über Ihre gesamte Entwicklungsumgebung.

**Einblicke ohne Ballast**
Schluss mit dem Rätselraten, welches Projekt noch ungesendete Commits oder vergessene Stashes hat. GitPulseMLC bietet detaillierte Zusammenfassungen Ihres Worktree-Status, inklusive geänderter Dateitypen und Branch-Aktualität. Es ist auf Geschwindigkeit ausgelegt und nutzt die Nebenläufigkeit von Go, um riesige Projektlisten mühelos zu verarbeiten.

**Bereit für Automatisierung**
Egal ob Sie einen schnellen Statuscheck im Terminal benötigen oder einen automatisierten HTML-Bericht für Ihr Team – GitPulseMLC bietet die passende Lösung. Mit JSON-Export und kontrastreicher HTML-Ausgabe integriert es sich perfekt in Ihre bestehenden Workflows und Monitoring-Skripte.

**Quickstart**
In wenigen Sekunden startklar mit Go. Kompilieren Sie den TUI-Client und starten Sie einen kompakten Scan, um nur die Repositories zu sehen, die Aufmerksamkeit erfordern:

```bash
# Binary erstellen
go build -o gitpulse ./cmd/tui

# Erster Scan im Kompakt-Modus
./gitpulse --compact
```
