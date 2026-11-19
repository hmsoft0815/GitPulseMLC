# GitPulseMLC

> **[mlcgo.eu](https://mlcgo.eu)** — Werkzeuge, Bibliotheken und Handbücher · [Produktseite](https://mlcgo.eu/products/gitpulse/) · [English](README.md)

**Ein Befehl zeigt dir, in welchem deiner lokalen Git-Repositories noch etwas offen ist:**
nicht committete Änderungen, nicht gepushte Commits, vergessene Stashes und verwaiste Branches —
und welche davon jeder im Internet lesen kann.
GitPulse liest nur — es committet, fetcht, pullt und pusht nie.
Einzige Netzwerkanfrage ist eine anonyme Prüfung, ob ein Remote öffentlich ist (`--offline` lässt sie weg).

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
- Du checkst **fremde Projekte** zum Ausprobieren aus und willst deren lokale Änderungen von deinen getrennt sehen.
- Du willst auf einen Blick sehen, welche Repositories **öffentlich** sind.

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

**2. Sag gitpulse, welche Repositories es beobachten soll**

```bash
gitpulse add ~/code           # jedes Git-Repository unterhalb von ~/code
gitpulse add                  # nur das Repository, in dem du gerade bist
```

Das schreibt `~/.gitpulsemlc/repos.ini`. Mit `--dry-run` siehst du vorher, was eingetragen würde.
[Projekte verwalten](#projekte-verwalten) erklärt Ignorieren, 3rd-party-Repositories und mehr.

**3. Hinschauen**

```bash
gitpulse                      # alle Repositories
gitpulse --compact            # nur die mit Handlungsbedarf
gitpulse -v                   # dazu jeder Branch mit Sync-Stand und Alter
```

## Projekte verwalten

```bash
gitpulse add ~/code              # alle Repositories unter einem Ordner (oder eins: gitpulse add PFAD)
gitpulse ignore altes-tool       # nicht mehr beobachten — per Name oder Pfad, z. B. `gitpulse ignore .`
gitpulse watch altes-tool        # wieder beobachten
gitpulse mark whisper.cpp 3rd    # Repository von jemand anderem (`own` macht es wieder zu deinem)
gitpulse remove altes-tool       # ganz vergessen (auf der Platte wird nichts angefasst)
gitpulse remove --missing        # alle Projekte vergessen, deren Ordner weg ist
gitpulse list                    # alle Projekte mit Art und Zustand
```

**Eigenes oder 3rd party.** Oft checkst du fremde Repositories aus, um sie anzusehen oder zu testen,
und hast dort lokale Änderungen, die nie upstream gehen sollen. Markiere sie als 3rd party:

- sie stehen in einem eigenen Block am Ende, mit `3rd` in der Spalte REMOTE;
- lokale Änderungen, lokale Commits, Stashes und alte Branches sind dort gedämpfte Hinweise, kein „Action required“;
- sie zählen nicht bei Dirty, Ahead, Behind oder Public, und `--compact` blendet sie aus;
- neue Commits upstream (`↓3`) bleiben sichtbar — das ist bei ihnen das Einzige, was zählt.

**Hat sich upstream etwas getan?** Das ist die Frage bei einem Repository, das du nur zum Testen geklont hast. gitpulse
fetcht nie, `↓3` zeigt also den Stand deines letzten `git fetch`. Fetche die 3rd-party-Checkouts, dann schau hin:

```bash
gitpulse --json | jq -r '.results[] | select(.third_party) | .path' |
  while read -r dir; do git -C "$dir" fetch --quiet; done
gitpulse --compact     # ein 3rd-party-Repository erscheint jetzt nur, wenn es upstream neue Commits gibt

# welche, und was ist neu
gitpulse --json | jq -r '.results[] | select(.third_party and .behind > 0) | "\(.name): \(.behind) neu — \(.incoming_commits[0])"'
```

Die ersten beiden Zeilen als wöchentlicher Cron-Job, und du merkst neue Versionen der Tools, die du gerade ausprobierst.

gitpulse sortiert neue Repositories selbst ein, sobald es weiß, welche Remotes deine sind:

```ini
[settings]
own_remotes = github.com/ich, gitlab.com/meine-gruppe, git.example.com
```

Ein Eintrag ist ein Host (`git.example.com`) oder Host und Owner (`github.com/ich`); `*` gilt innerhalb eines Abschnitts.
Ein Repository ist deins, wenn ein Remote passt, ein Remote im lokalen Netz liegt oder es gar kein Remote hat.
Solange `own_remotes` fehlt, gilt alles als deins, und `gitpulse add` schlägt die gefundenen Remote-Owner vor.
`gitpulse mark --auto` sortiert die Projekte, die du schon hast (vorher mit `--dry-run` ansehen).

**Ignorierte** Projekte bleiben gemerkt, `gitpulse add` holt sie also nie zurück.
`gitpulse ignore PFAD` geht auch für ein Repository, das noch nicht in der Config steht.
**Entfernte** Projekte verschwinden ganz aus der Config; ein späteres `gitpulse add` des Elternordners fände sie wieder.
`remove` ist für Repositories, die es nicht mehr gibt — `gitpulse remove --missing` (vorher mit `--dry-run` ansehen) vergisst
alle Projekte, deren Ordner weg ist oder kein Git-Repository mehr enthält; `gitpulse list` zeigt sie als `missing`.

Die Config hat je Zustand eine Section — reines ini, auch von Hand bearbeitbar:

```ini
[projects]           # eigene
webshop = /home/ich/code/webshop

[third_party]        # zum Ansehen oder Testen ausgecheckt
whisper.cpp = /home/ich/src/whisper.cpp

[disabled_projects]  # ignoriert
altes-tool = /home/ich/code/altes-tool
```

Jeder Name darf nur einmal vorkommen. Steht ein Name doppelt drin, wird nur der letzte Eintrag beobachtet; gitpulse warnt, und `gitpulse list` markiert den anderen als `dup`.

## Typische Einsätze

```bash
# Morgens: Was habe ich gestern offen gelassen?
gitpulse --compact

# Tägliche Status-Mail per Cron
0 8 * * 1-5  gitpulse --compact --html | mail -a "Content-Type: text/html" -s "Repo-Status" ich@example.com

# Ist etwas öffentlich, das es nicht sein sollte?
gitpulse --public

# Geklonte Tools zum Ausprobieren: Hat sich upstream etwas getan? (siehe Projekte verwalten)
gitpulse --json | jq -r '.results[] | select(.third_party) | .path' | while read -r d; do git -C "$d" fetch -q; done; gitpulse --compact

# Eigenes Monitoring oder Skripte füttern (eigene Projekte mit offenen Änderungen)
gitpulse --json | jq -r '.results[] | select(.third_party | not) | select(.is_clean == false) | .name'

# Aufräumen: Projekte vergessen, deren Ordner weg ist
gitpulse remove --missing --dry-run

# Repositories auf einem Server per SSH prüfen (siehe Remote-Monitoring)
remote_gitpulse -host user@server -config server-repos.ini -arch linux-amd64
```

---

## Referenz

### Kommandozeile

```
gitpulse [flags]                       Status aller beobachteten Repositories
gitpulse add [flags] [PFAD|DIR ...]    ein Repository eintragen, oder alle unter einem Ordner (Alias: init)
gitpulse ignore NAME|PFAD ...          nicht mehr beobachten, aber merken
gitpulse watch NAME|PFAD ...           ignoriertes Projekt wieder beobachten
gitpulse mark NAME|PFAD ... own|3rd    eigenes Projekt oder 3rd party
gitpulse mark --auto [--dry-run]       eigene Projekte nach own_remotes sortieren
gitpulse remove NAME|PFAD ...          Projekte vergessen (Alias: rm)
gitpulse remove --missing [--dry-run]  alle Projekte vergessen, deren Ordner weg ist
gitpulse list                          alle Projekte mit Art und Zustand
gitpulse BEFEHL --help                 Flags eines Befehls

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

add-Flags:
  --depth N      wie viele Verzeichnisebenen durchsucht werden (Standard 4)
  --dry-run      nur auflisten, was eingetragen würde
  --own, --3rd   als eigenes / 3rd party eintragen, egal welches Remote
  --config PATH  Config-Datei, die angelegt oder erweitert wird
```

`add` überspringt versteckte Ordner, `node_modules` und `vendor` und schaut nicht in ein bereits gefundenes Repository hinein.
Projekte, die schon in der Config stehen — beobachtet oder ignoriert —, werden nie doppelt eingetragen. Flags dürfen auch hinter den Pfaden stehen.

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
# deine Remotes; Repositories anderswo werden als 3rd party eingetragen
own_remotes = github.com/ich, git.example.com

[third_party]
whisper.cpp = /home/ich/src/whisper.cpp

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
task install      # bauen und für deinen Benutzer installieren: ~/.local/bin/gitpulse + Manpage (Linux, macOS)
task test
task --list       # alle Tasks
```

`task install PREFIX=/opt/tools` installiert woanders hin; `task uninstall` entfernt es wieder und lässt deine Config in Ruhe.
Die Tasks unter `maint:` signieren und laden die Release-Downloads hoch; sie brauchen den Signier-Mac und den Download-Server
der Maintainer und brechen überall sonst mit einer Erklärung ab — `task release:snapshot` baut alle Release-Artefakte lokal.

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
