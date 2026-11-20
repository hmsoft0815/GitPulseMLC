# GitPulseMLC Go Library Documentation

The core Git scanning and monitoring logic of GitPulseMLC is organized into the `pkg/` directory, making it a fully reusable Go library. You can integrate it into your own applications, web dashboards, or background services.

## Installation

```bash
go get github.com/hmsoft0815/GitPulseMLC
```

## Core Packages

### 1. `gitmonitor`
This is the main package for scanning repositories. It asks the installed `git` binary when there is one and falls back to the built-in `go-git` otherwise. `NewScannerWithOptions(gitmonitor.Options{Backend: gitmonitor.BackendGoGit})` never calls `git`; `BackendGit` and `BackendAuto` (the default) do.

#### Key Features:
*   **Dirty State Detection**: Identify modified, untracked, and deleted files.
*   **Branch Analysis**: Calculate ahead/behind counts for all local branches.
*   **Staleness Tracking**: Identify branches that haven't been updated in 30+ days.
*   **File Type Grouping**: Summarize changes by file extension (e.g., `.go`, `.md`).

#### Basic Usage:

```go
import "github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"

func example() {
    scanner := gitmonitor.NewScanner()
    
    // Status contains everything: branch info, sync counts, stashes, etc.
    status, _ := scanner.Scan("my-project", "/abs/path/to/repo")
    
    if status.Error != nil {
        fmt.Printf("Scan failed: %v\n", status.Error)
        return
    }

    fmt.Printf("Current Branch: %s\n", status.CurrentBranch)
    fmt.Printf("Clean: %v\n", status.IsClean)
}
```

To scan many repositories, `ScanAll` runs scans in parallel and keeps the order of its input:

```go
targets := []gitmonitor.Target{{Name: "api", Path: "/code/api"}, {Name: "web", Path: "/code/web"}}
results := gitmonitor.ScanAll(gitmonitor.NewScanner(), targets, 0, nil) // 0 = one worker per CPU
```

### 2. `config`
A helper package to load and manage `repos.ini` configuration files.

```go
import "github.com/hmsoft0815/GitPulseMLC/pkg/config"

func load() {
    cfg, _ := config.LoadConfig("repos.ini")
    for name, path := range cfg.Projects {
        // ...
    }
}
```

To add projects programmatically (creating the file if needed), use `config.NewManager(path).AddProjects(...)`;
`Known()` returns the names and paths already listed. `Entries()` lists every project with its section
(`config.SectionOwn`, `SectionThirdParty`, `SectionIgnored`), `Lookup(nameOrPath)` finds one, and
`Move(name, section)` ignores, watches or re-marks it while keeping comments in the file.
`Config.ThirdParty` tells which watched projects are third party.

### 3. `discover`
Finds Git repositories below a directory — the engine behind `gitpulse add`.

```go
import "github.com/hmsoft0815/GitPulseMLC/pkg/discover"

func find() {
    paths, err := discover.Find("/home/me/code", discover.DefaultMaxDepth)
    if err != nil {
        return
    }
    for _, r := range discover.Name(paths, map[string]bool{}) {
        fmt.Println(r.Name, r.Path) // unique project names, e.g. "client-api"
    }
}
```

### 4. `visibility`

Tells whether a Git remote can be read by anyone. Remotes on the local network (file paths, hosts without a dot, `.local`/`.lan`, private IPs, configured patterns) are `local` without any network access; for other hosts an anonymous HTTPS request to the smart HTTP endpoint decides between `public` and `private`. Results are cached.

```go
import "github.com/hmsoft0815/GitPulseMLC/pkg/visibility"

checker := visibility.NewChecker(visibility.Options{
    PrivateHosts: []string{"git.example.com"},
    CachePath:    visibility.DefaultCachePath(),
})
res := checker.Check(ctx, "git@github.com:owner/repo.git")
fmt.Println(res.Host, res.Visibility) // github.com public
_ = checker.Save()
```

The scanner fills `RepoStatus.Remotes` with names and credential-free URLs; it does not touch the network. Run a `Checker` over them to fill `Host` and `Visibility`.

### 5. `ownership`

Decides whether a repository is yours or third party, from its remotes and your `own_remotes` patterns.

```go
import "github.com/hmsoft0815/GitPulseMLC/pkg/ownership"

c := &ownership.Classifier{OwnRemotes: []string{"github.com/me", "git.example.com"}}
urls, _ := ownership.RemoteURLs("/home/me/src/whisper.cpp")
fmt.Println(c.Classify(urls)) // "3rd" — no remote of yours, none on the local network
```

## JSON Export

All models in `pkg/gitmonitor` are equipped with standard `json` struct tags. You can easily generate the same data format used by the CLI:

```go
import (
    "encoding/json"
    "github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

func toJson(status *gitmonitor.RepoStatus) string {
    bytes, _ := json.MarshalIndent(status, "", "  ")
    return string(bytes)
}
```

## Data Models

Refer to the [JSON Specification](./json_schema.md) for a detailed description of the fields available in `RepoStatus`, `BranchInfo`, and `Summary`.
