# GitPulseMLC

> **[mlcgo.eu](https://mlcgo.eu)** — Werkzeuge, Bibliotheken und Handbücher · [Produktseite](https://mlcgo.eu/products/gitpulse/) · [English](README.md)

**Ein Befehl zeigt dir, in welchem deiner lokalen Git-Repositories noch etwas offen ist:**
nicht committete Änderungen, nicht gepushte Commits, vergessene Stashes und verwaiste Branches.
GitPulse liest nur — es committet, pullt und pusht nie.

```
$ gitpulse
NAME          BRANCH   REMOTE         STATUS               PATH
──────────────────────────────────────────────────────────────────────────────
api-server    main     gitlab priv    ↑2                   ~/code/api-server
blog          main     github pub     CLEAN 1 stale        ~/code/blog
dotfiles      main     —              CLEAN                ~/code/dotfiles
invoice-tool  main     nas.local lan  1 stash              ~/code/invoice-tool
webshop       main     github pub +1  DIRTY (1.css, 2.go)  ~/code/webshop

SUMMARY:
  Total Projects: 5
  Dirty:         [ 1 ] (Action required!)
  Ahead:         [ 1 ] (Need to push)
  Public:        [ 2 ] (readable by anyone)
```

So liest du die Statusspalte:

| Status | Bedeutung | Was tun |
|---|---|---|
| `DIRTY (2.go)` | Nicht committete Änderungen, nach Dateityp gruppiert | committen oder verwerfen |
| `↑2` | 2 lokale Commits, noch nicht gepusht | `git push` |
| `↓3` | 3 Commits auf dem Remote, die dir fehlen | `git pull` |
| `1 stash` | Mit `git stash` geparkte Arbeit | anwenden oder löschen |
| `SUB .mlcai` | Ein Submodule hat selbst nicht committete, nicht gepushte oder geparkte Arbeit | im Submodule erledigen |
| `1 stale` | Ein lokaler Branch, seit über 30 Tagen unberührt | löschen oder fertigmachen |
| `1 sub-ptr` | Ein Submodule ist über den im Eltern-Repo eingetragenen Commit hinaus; nichts geht verloren | neuen Zeiger bei Gelegenheit committen |

So liest du die Remote-Spalte — wer das Repository lesen kann:

| Remote | Bedeutung |
|---|---|
| `github pub` | Jeder kann es ohne Anmeldung lesen — prüfen, dass keine Secrets oder internen Inhalte drin sind |
| `gitlab priv` | Externer Host, der anonymes Lesen ablehnt (privat oder GitLab „internal“) |
| `nas.local lan` | Pfad oder Host im lokalen Netz: per Definition privat, wird nie online geprüft |
| `github ?` | Unbekannt: `--offline` ohne Ergebnis im Cache, oder der Host war nicht erreichbar |
| `—` | Kein Remote: das Repository existiert nur auf diesem Rechner |
| `… +1` | Ein weiteres Remote; die Spalte zeigt das offenste, `--all` listet alle |

Lokales Netz heißt: ein Dateipfad, ein Hostname ohne Punkt, `.local`, `.lan`, `.home`, `.internal`,
`.fritz.box`, eine private IP oder ein Host aus `private_hosts`.
Bei allen anderen Hosts fragt gitpulse anonym per HTTPS die Refs des Repositories ab
(`/info/refs?service=git-upload-pack`) — dieselbe Anfrage wie `git clone`, ohne deine Zugangsdaten,
Git-Config oder Credential-Helper. SSH-Aliase aus `~/.ssh/config` werden vorher aufgelöst.
Das Ergebnis wird 7 Tage im Cache-Verzeichnis des Benutzers gehalten (unter Linux `~/.cache/gitpulse/visibility.json`).

## Ist das was für mich?

- Du arbeitest an **10 oder mehr Repositories** auf einem Rechner oder Server.
- Dir ist schon Arbeit verloren gegangen, weil ein Commit **nie gepusht** wurde.
- Du willst einen täglichen **„Was ist noch offen?“-Bericht** per Mail, Cron oder CI.

Wer nur in einem Repository arbeitet, kommt mit `git status` aus.

## Schnellstart

**1. Installieren**

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/hmsoft0815/GitPulseMLC/main/install.sh | sh

# oder mit Go
go install github.com/hmsoft0815/GitPulseMLC/cmd/gitpulse@latest
```

Pakete für Debian/Ubuntu (`.deb`), Fedora (`.rpm`), Alpine (`.apk`) sowie Archive für Linux und Windows gibt es auf der
[Release-Seite](https://github.com/hmsoft0815/GitPulseMLC/releases). **macOS**-Binaries — signiert und notarisiert — gibt es auf
[mlcgo.eu](https://mlcgo.eu/products/gitpulse/); `install.sh` holt sie von dort.

**2. Sag gitpulse, wo deine Repositories liegen**

```bash
gitpulse init ~/code          # findet jedes Git-Repository unterhalb von ~/code
gitpulse init ~/work          # nochmal aufrufen, um weitere Ordner hinzuzufügen
```

Das schreibt `~/.gitpulsemlc/repos.ini`. Mit `--dry-run` siehst du vorher, was eingetragen würde.

**3. Hinschauen**

```bash
gitpulse                      # alle Repositories
gitpulse --compact            # nur die mit Handlungsbedarf
gitpulse -v                   # dazu jeder Branch mit Sync-Stand und Alter
```

## Typische Einsätze

```bash
# Morgens: Was habe ich gestern offen gelassen?
gitpulse --compact

# Tägliche Status-Mail per Cron
0 8 * * 1-5  gitpulse --compact --html | mail -a "Content-Type: text/html" -s "Repo-Status" ich@example.com

# Eigenes Monitoring oder Skripte füttern
gitpulse --json | jq -r '.results[] | select(.is_clean == false) | .name'

# Repositories auf einem Server per SSH prüfen (siehe Remote-Monitoring)
remote_gitpulse -host user@server -config server-repos.ini -arch linux-amd64
```

---

## Referenz

### Kommandozeile

```
gitpulse [flags]                 Status aller konfigurierten Repositories
gitpulse init [flags] [DIR ...]  Repositories unter DIR finden und in die Config eintragen

Flags:
  --compact      nur Repositories mit Handlungsbedarf
  -v, --all      Details zu jedem Remote, lokalen Branch, Stash und Submodule
  --jobs N       so viele Repositories parallel scannen (Standard: eins pro CPU)
  --public       nur Repositories mit öffentlich lesbarem Remote
  --offline      nicht online prüfen, ob Remotes öffentlich sind; Ergebnisse aus dem Cache nehmen
  --progress     Scan-Fortschritt anzeigen (nur Terminal-Ausgabe)
  --json         JSON-Ausgabe (Schema: docs/json_schema.md)
  --html         kontrastreicher HTML-Bericht, z. B. für E-Mails
  --no-color     reiner Text
  --config PATH  diese Config-Datei verwenden
  --version      Version ausgeben

init-Flags:
  --depth N      wie viele Verzeichnisebenen durchsucht werden (Standard 4)
  --dry-run      nur auflisten, was eingetragen würde
  --config PATH  Config-Datei, die angelegt oder erweitert wird
```

`init` überspringt versteckte Ordner, `node_modules` und `vendor` und schaut nicht in ein bereits gefundenes Repository hinein.
Projekte, die schon in der Config stehen — aktiv oder deaktiviert —, werden nie doppelt eingetragen.

### Konfiguration

Suchreihenfolge: `--config`, `$GITPULSE_CONFIG`, `./config/repos.ini`, `~/.gitpulsemlc/repos.ini`.

```ini
[projects]
# Name = absoluter Pfad
webshop = /home/ich/code/webshop
api-server = /home/ich/code/api-server

[general]
# wird in der Ausgabe als "~" angezeigt
replace_path_prefix = /home/ich
replace_with = ~

[settings]
column_name_width = 25
show_summary = true
compact_mode = false
# Hosts, die privat sind, obwohl sie extern aussehen (Glob-Muster)
private_hosts = git.example.com, *.corp.example

[disabled_projects]
# hierher verschoben: wird nicht gescannt, bleibt aber gemerkt
altes-tool = /home/ich/code/altes-tool
```

Ein vollständig kommentiertes Beispiel liegt in [`config/repos.ini.example`](config/repos.ini.example).

### Remote-Monitoring

`remote_gitpulse` kopiert das passende gitpulse-Binary und deine Config per SSH auf einen Linux- oder macOS-Server,
führt es dort aus und räumt danach auf.

```bash
task build:cross build:remote
./bin/remote_gitpulse -host user@server -config server-repos.ini -arch linux-amd64
```

### Go-Bibliothek

Der Scanner ist ein normales Go-Paket, das du in eigenen Werkzeugen nutzen kannst:

```bash
go get github.com/hmsoft0815/GitPulseMLC
```

Siehe [docs/LIBRARY.md](docs/LIBRARY.md).

### Aus dem Quellcode bauen

Benötigt Go 1.27.2+ und [Task](https://taskfile.dev).

```bash
task build        # bin/gitpulse
task test
task --list       # alle Tasks
```

> **Windows:** Der Defender kann das unsignierte Binary blockieren, weil es schnell viele Dateien liest.
> In dem Fall eine Ausnahme für `gitpulse.exe` anlegen.

---

## Warum GitPulseMLC?

Bei 20 und mehr aktiven Repositories auf einem Rechner oder Server gehen Commits vergessen und Branches verwaisen.
GUI-Werkzeuge sind schwer und verlangen, jedes Projekt einzeln anzuklicken. GitPulse gibt eine Antwort — so schnell
wie `ls` und so aussagekräftig wie `git status`, für alle auf einmal.

### Credits

- **Engine:** [go-git](https://github.com/go-git/go-git)
- **UI:** [Lip Gloss](https://github.com/charmbracelet/lipgloss)

## Lizenz

Copyright (c) 2026 Michael Lechner. MIT-Lizenz — siehe [LICENSE](LICENSE). Änderungen: [CHANGELOG.md](CHANGELOG.md).

<!-- mlcai-private -->
## Projektdokumentation (`.mlcai/`)

`.mlcai/` enthält interne Planung, Backlog und Arbeitsnotizen, gepflegt mit dem MLC Doc Hub.
Es ist ein separater privater Klon, nicht Teil dieses Repositories, und für den Build nicht nötig.
