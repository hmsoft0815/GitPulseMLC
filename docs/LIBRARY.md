# GitPulseMLC Go Library Documentation

The core Git scanning and monitoring logic of GitPulseMLC is organized into the `pkg/` directory, making it a fully reusable Go library. You can integrate it into your own applications, web dashboards, or background services.

## Installation

```bash
go get github.com/hmsoft0815/GitPulseMLC
```

## Core Packages

### 1. `gitmonitor`
This is the main package for scanning repositories. It leverages `go-git` to perform deep analysis of local repositories without requiring the `git` binary.

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
