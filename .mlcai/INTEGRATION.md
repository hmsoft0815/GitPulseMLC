# 🤖 AI & Integration Context: GitPulseMLC

## 1. Identität & Zweck
- **Kernaufgabe:** Hochperformantes Dashboard zur Überwachung lokaler Git-Repositories.
- **Technischer Stack:** Go, Lip Gloss (TUI), go-git.
- **Hoster/Infrastruktur:** Lokale Ausführung als Binary; Remote-Execution via SSH möglich.

## 2. Die "Nachbarschaft" (System-Kontext)
- **Upstream (Wovon hänge ich ab?):**
  - **Lokal installiertes Git:** Für einige erweiterte Informationen (optional, primär nutzt es go-git).
- **Downstream (Wer nutzt mich?):**
  - **Entwickler:** Nutzen das TUI für den täglichen Überblick.
  - **CI/CD / Monitoring:** Nutzen den JSON-Export (`--json`) für automatisierte Status-Checks.
  - **E-Mail-Services:** Nutzen den HTML-Report (`--html`) für regelmäßige Statusberichte per E-Mail.
- **Shared Resources:**
  - Nutzt das globale MLC-Glossar für Begriffe wie "User" und "System".

## 3. Schnittstellen-Vertrag
- **Primäre Schnittstelle:** CLI (Befehlszeilenparameter).
- **Output-Formate:**
  - **TUI:** Interaktiv, farbig, für Menschen optimiert.
  - **JSON:** Strukturiert, für Maschinen (Schema in `docs/schema.json`).
  - **HTML:** Formatiert, für E-Mail-Clients optimiert.
- **Wichtige Datenmodelle:**
  - `RepoStatus`: { Name, Path, CurrentBranch, IsClean, Ahead, Behind, ChangedFiles, ErrorMsg }
- **Konfiguration:** `repos.ini` (Look-up: `--config`, `GITPULSE_CONFIG`, `./config/repos.ini`, `~/.gitpulsemlc/repos.ini`).

## 4. Leitplanken & Regeln
- **Naming:** CamelCase für Go-Variablen/Structs, Kebab-Case für CLI-Flags.
- **Testing:** "Jeder neue Scanner-Feature erfordert Tests in `pkg/gitmonitor/scanner_test.go`."
- **Performance:** Scannen von 100+ Repositories sollte < 2 Sekunden dauern (Go Concurrency).

## 5. Aktueller Fokus (Status)
- **Bekannte Probleme:** Performance-Optimierung bei sehr tiefen Verzeichnisstrukturen (Pfad-Kürzung aktiv).
- **Nächste Schritte:** Erweiterung der Remote-Execution-Funktionalität und Verfeinerung des HTML-Reports.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI Agent
- **Status:** Aktuell
