# 🛠 Tech Stack & Constraints: GitPulseMLC

## Kern-Versionen
- **Sprache:** Go 1.25+
- **Architektur:** Monolithisches CLI-Tool mit modularer Scan-Engine (`pkg/gitmonitor`).

## Bibliotheken (Erlaubt/Fixiert)
- **Git Engine:** `github.com/go-git/go-git/v5` (Native Go Git Implementation).
- **UI (TUI):** `github.com/charmbracelet/lipgloss` für modernes Terminal-Styling und Layout.
- **Terminal Features:** `github.com/muesli/termenv` für Farberkennung und High-Level-Terminal-Features.
- **Konfiguration:** `gopkg.in/ini.v1` für das Einlesen der `repos.ini`.

## Einschränkungen (Constraints)
- **Read-Only:** Das Tool darf unter keinen Umständen Schreibzugriffe auf Git-Repositories ausführen (kein `commit`, `push`, `pull` oder `merge`).
- **Keine externen Abhängigkeiten:** Das Tool muss ohne laufende Datenbanken oder externe APIs funktionieren (reine lokale Analyse).
- **Cross-Platform:** Alle Pfadoperationen müssen via `path/filepath` (Go) gehandhabt werden, um volle Kompatibilität mit Windows/Linux/macOS zu gewährleisten.
- **Keine UI-Klassen:** Nur funktionale Programmierung und Struct-basierte Datenmodelle nutzen (idiomatisches Go).

## Styling-Regeln
- **TUI-Design:** Dunkles Theme bevorzugt, Lip Gloss für konsistente Farbschemata (Dirty: Orange/Gelb, Ahead: Blau, Error: Rot).
- **HTML-Reports:** High-Contrast-Design für gute Lesbarkeit in E-Mails und Dokumentationen.
- **Code-Stil:** Variablen auf Englisch, Kommentare auf Englisch (Standard im Projekt).
- **Concurrency:** Go-Routines für paralleles Scannen großer Repository-Listen nutzen (Standard in `scanner.go`).

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI Agent
- **Status:** Aktuell
