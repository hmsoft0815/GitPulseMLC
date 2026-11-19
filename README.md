# GitPulseMLC

> **[mlcgo.eu](https://mlcgo.eu)** — tools, libraries and manuals · [Product page](https://mlcgo.eu/products/gitpulse/) · [Deutsch](README-de.md)

**One command shows which of your local Git repositories still need attention:**
uncommitted changes, unpushed commits, forgotten stashes and stale branches.
GitPulse only reads your repositories — it never commits, pulls or pushes.

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

Reading the status column:

| Status | Meaning | What to do |
|---|---|---|
| `DIRTY (2.go)` | Uncommitted changes, grouped by file type | commit or discard |
| `↑2` | 2 local commits not pushed yet | `git push` |
| `↓3` | 3 commits on the remote you don't have | `git pull` |
| `1 stash` | Work parked with `git stash` | apply or drop it |
| `SUB .mlcai` | A submodule has uncommitted, unpushed or stashed work of its own | fix it inside the submodule |
| `1 stale` | A local branch untouched for 30+ days | delete or finish it |
| `1 sub-ptr` | A submodule moved past the commit the parent records; nothing is lost | commit the new pointer when convenient |

Reading the remote column — who can read the repository:

| Remote | Meaning |
|---|---|
| `github pub` | Anyone can read it without logging in — check that no secrets or internal code are in it |
| `gitlab priv` | External host that refused an anonymous read (private, or GitLab "internal") |
| `nas.local lan` | Path or host on your local network: private by definition, never checked online |
| `github ?` | Not known: `--offline` without a cached result, or the host was unreachable |
| `—` | No remote: the repository exists only on this machine |
| `… +1` | One more remote; the column shows the most open one, `--all` lists them all |

Local network means: a file path, a host name without a dot, `.local`, `.lan`, `.home`, `.internal`,
`.fritz.box`, a private IP, or a host listed in `private_hosts`.
For every other host, gitpulse asks anonymously for the repository's refs over HTTPS
(`/info/refs?service=git-upload-pack`) — the same request `git clone` makes, without your credentials,
Git config or credential helpers. SSH aliases from `~/.ssh/config` are resolved first.
Results are cached for 7 days in the user cache directory (`~/.cache/gitpulse/visibility.json` on Linux).

## Is this for me?

- You work on **10+ repositories** on one machine or server.
- You have lost work because a commit **never got pushed** before switching laptops.
- You want a daily **"what's still open?" report** by mail, cron or CI.

If you only ever work in one repository, `git status` is enough.

## Quickstart

**1. Install**

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/hmsoft0815/GitPulseMLC/main/install.sh | sh

# or with Go
go install github.com/hmsoft0815/GitPulseMLC/cmd/gitpulse@latest
```

Debian/Ubuntu (`.deb`), Fedora (`.rpm`), Alpine (`.apk`) packages and Linux/Windows archives are on the
[releases page](https://github.com/hmsoft0815/GitPulseMLC/releases). **macOS** binaries — signed and notarized — are on
[mlcgo.eu](https://mlcgo.eu/products/gitpulse/); `install.sh` fetches them from there.

**2. Tell gitpulse where your repositories are**

```bash
gitpulse init ~/code          # finds every Git repository below ~/code
gitpulse init ~/work          # run again to add more folders
```

This writes `~/.gitpulsemlc/repos.ini`. Use `--dry-run` to see what would be added first.

**3. Look**

```bash
gitpulse                      # all repositories
gitpulse --compact            # only the ones that need attention
gitpulse -v                   # plus every branch, its sync state and age
```

## Typical uses

```bash
# Morning check: what did I leave open yesterday?
gitpulse --compact

# Daily status mail via cron
0 8 * * 1-5  gitpulse --compact --html | mail -a "Content-Type: text/html" -s "Repo status" me@example.com

# Feed your own monitoring or scripts
gitpulse --json | jq -r '.results[] | select(.is_clean == false) | .name'

# Check repositories on a server over SSH (see Remote monitoring)
remote_gitpulse -host user@server -config server-repos.ini -arch linux-amd64
```

---

## Reference

### Command line

```
gitpulse [flags]                 show the status of all configured repositories
gitpulse init [flags] [DIR ...]  find repositories below DIR and add them to the config

Flags:
  --compact      only repositories that need action
  -v, --all      details for every remote, local branch, stash and submodule
  --jobs N       repositories scanned in parallel (default: one per CPU)
  --public       only repositories with a publicly readable remote
  --offline      do not check online whether remotes are public; use cached results
  --progress     show scanning progress (terminal output only)
  --json         JSON output (schema: docs/json_schema.md)
  --html         high-contrast HTML report, e.g. for e-mail
  --no-color     plain text
  --config PATH  use this config file
  --version      print the version

init flags:
  --depth N      how many directory levels to search (default 4)
  --dry-run      only list what would be added
  --config PATH  config file to create or extend
```

`init` skips hidden folders, `node_modules` and `vendor`, and does not look inside a repository it has found.
Projects already in the config — enabled or disabled — are never added twice.

### Configuration

Lookup order: `--config`, `$GITPULSE_CONFIG`, `./config/repos.ini`, `~/.gitpulsemlc/repos.ini`.

```ini
[projects]
# name = absolute path
webshop = /home/me/code/webshop
api-server = /home/me/code/api-server

[general]
# shown as "~" in the output
replace_path_prefix = /home/me
replace_with = ~

[settings]
column_name_width = 25
show_summary = true
compact_mode = false
# hosts that are private although they look external (glob patterns)
private_hosts = git.example.com, *.corp.example

[disabled_projects]
# moved here: not scanned, but remembered
old-tool = /home/me/code/old-tool
```

A fully commented example is in [`config/repos.ini.example`](config/repos.ini.example).

### Remote monitoring

`remote_gitpulse` copies the matching gitpulse binary and your config to a Linux or macOS server via SSH,
runs it there and cleans up afterwards.

```bash
task build:cross build:remote
./bin/remote_gitpulse -host user@server -config server-repos.ini -arch linux-amd64
```

### Go library

The scanner is a plain Go package you can use in your own tools:

```bash
go get github.com/hmsoft0815/GitPulseMLC
```

See [docs/LIBRARY.md](docs/LIBRARY.md).

### Building from source

Requires Go 1.27.2+ and [Task](https://taskfile.dev).

```bash
task build        # bin/gitpulse
task test
task --list       # all tasks
```

> **Windows:** Defender may block the unsigned binary because it reads many files quickly.
> Add an exclusion for `gitpulse.exe` if that happens.

---

## Why GitPulseMLC?

With 20+ active repositories on a dev machine or server, commits get forgotten and branches go stale.
GUI tools are heavy and need clicking through each project. GitPulse gives one answer, as fast as `ls`
and as informative as `git status` — for all of them at once.

### Credits

- **Engine:** [go-git](https://github.com/go-git/go-git)
- **UI:** [Lip Gloss](https://github.com/charmbracelet/lipgloss)

## License

Copyright (c) 2026 Michael Lechner. MIT License — see [LICENSE](LICENSE). Changes: [CHANGELOG.md](CHANGELOG.md).

<!-- mlcai-private -->
## Project documentation (`.mlcai/`)

`.mlcai/` holds internal planning, backlog and work notes, maintained with the MLC Doc Hub.
It is a separate private clone, not part of this repository, and not needed to build.

## Who is "Claude" in the commits?

Some commits in this repository are co-authored by Claude, Anthropic's AI
model. It helps write code, keeps our documentation and backlog up to date and
digs through failing builds — every change is reviewed before it is merged.
We don't hide it: [how we work with Claude](https://mlcgo.eu/ai/en.html).
