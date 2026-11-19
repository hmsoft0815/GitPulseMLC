// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/render"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
	"github.com/hmsoft0815/GitPulseMLC/pkg/visibility"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// version is set via ldflags during build (e.g. by GoReleaser)
var version = "dev"

func findConfig(customPath string) string {
	if customPath != "" {
		return customPath
	}
	if envPath := os.Getenv("GITPULSE_CONFIG"); envPath != "" {
		return envPath
	}
	localConfig := filepath.Join("config", "repos.ini")
	if _, err := os.Stat(localConfig); err == nil {
		return localConfig
	}
	home, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(home, ".gitpulsemlc", "repos.ini")
		if _, err := os.Stat(homeConfig); err == nil {
			return homeConfig
		}
	}
	return ""
}

// commands are the project subcommands; init is the old name of add.
var commands = map[string]func(args []string, globalConfig string, out io.Writer) error{
	"add":    runAdd,
	"init":   runAdd,
	"ignore": runIgnore,
	"watch":  runWatch,
	"mark":   runMark,
	"list":   runList,
	"remove": runRemove,
	"rm":     runRemove,
}

func main() {
	// CLI Flags
	configFlag := flag.String("config", "", "Path to the configuration file (repos.ini)")
	showDetails := flag.Bool("v", false, "Show details for all local branches")
	showAll := flag.Bool("all", false, "Show details for all local branches")
	noColor := flag.Bool("no-color", false, "Disable color output")
	compactFlag := flag.Bool("compact", false, "Only show repos needing action")
	printVersion := flag.Bool("version", false, "Print version information")
	jsonOutput := flag.Bool("json", false, "Output results in JSON format")
	htmlOutput := flag.Bool("html", false, "Output results in HTML format (suitable for email)")
	showProgress := flag.Bool("progress", false, "Show scanning progress (TUI mode only)")
	jobs := flag.Int("jobs", 0, "Repositories to scan in parallel (default: one per CPU)")
	offline := flag.Bool("offline", false, "Do not check online whether remotes are public; use cached results")
	publicOnly := flag.Bool("public", false, "Only show repositories with a publicly readable remote")
	layoutFlag := flag.String("layout", "", "Table layout: auto (fit the terminal), wide, narrow or stacked")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "GitPulseMLC  - Monitor the heartbeat of your local Git repositories\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  gitpulse [flags]                     show the status of all watched repositories\n")
		fmt.Fprintf(os.Stderr, "  gitpulse add [PATH|DIR ...]          add a repository, or all below a folder (alias: init)\n")
		fmt.Fprintf(os.Stderr, "  gitpulse ignore NAME|PATH ...        stop watching, but remember\n")
		fmt.Fprintf(os.Stderr, "  gitpulse watch NAME|PATH ...         watch an ignored project again\n")
		fmt.Fprintf(os.Stderr, "  gitpulse mark NAME|PATH ... own|3rd  your own project or third party (--auto: by own_remotes)\n")
		fmt.Fprintf(os.Stderr, "  gitpulse remove NAME|PATH ...        forget projects (--missing: all whose folder is gone)\n")
		fmt.Fprintf(os.Stderr, "  gitpulse list                        all projects with kind and state\n")
		fmt.Fprintf(os.Stderr, "  gitpulse COMMAND --help              flags of a command\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConfig Lookup Order:\n")
		fmt.Fprintf(os.Stderr, "  1. --config flag\n")
		fmt.Fprintf(os.Stderr, "  2. GITPULSE_CONFIG environment variable\n")
		fmt.Fprintf(os.Stderr, "  3. config/repos.ini (current directory)\n")
		fmt.Fprintf(os.Stderr, "  4. ~/.gitpulsemlc/repos.ini (user home)\n")
	}

	flag.Parse()

	if run, ok := commands[flag.Arg(0)]; ok {
		if err := run(flag.Args()[1:], *configFlag, os.Stdout); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "gitpulse %s: %v\n", flag.Arg(0), err)
			os.Exit(1)
		}
		return
	}
	if flag.NArg() > 0 {
		log.Fatalf("unknown command %q — see gitpulse --help", flag.Arg(0))
	}

	if *printVersion {
		fmt.Printf("GitPulseMLC version %s\n", version)
		os.Exit(0)
	}

	if *noColor {
		lipgloss.SetColorProfile(termenv.Ascii)
	}

	details := *showDetails || *showAll

	cfg := loadConfigOrExplain(*configFlag)

	if *compactFlag {
		cfg.Settings.CompactMode = true
	}

	layout, err := chooseLayout(*layoutFlag, cfg.Settings.Layout)
	if err != nil {
		log.Fatal(err)
	}
	isReportMode := *jsonOutput || *htmlOutput
	allResults := scanProjects(cfg.Projects, *jobs, *showProgress && !isReportMode)
	markThirdParty(allResults, cfg.ThirdParty)
	checkVisibility(allResults, cfg.Settings, *offline)
	if *publicOnly {
		allResults = onlyPublic(allResults)
	}
	summary := summarize(allResults)

	// 2. Output processing
	if *jsonOutput {
		if err := render.JSON(os.Stdout, version, allResults, summary); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *htmlOutput {
		if err := render.HTML(os.Stdout, version, allResults, summary, !*noColor, cfg.Settings); err != nil {
			log.Fatal(err)
		}
		return
	}

	render.TUI(os.Stdout, allResults, summary, cfg, render.TUIOptions{Details: details, Layout: layout, Width: terminalWidth()})
}

// scanProjects scans all projects in parallel and returns them sorted by name.
func scanProjects(projects map[string]string, jobs int, progress bool) []*gitmonitor.RepoStatus {
	targets := make([]gitmonitor.Target, 0, len(projects))
	for name, path := range projects {
		targets = append(targets, gitmonitor.Target{Name: name, Path: path})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Name < targets[j].Name })

	var onDone func(int, gitmonitor.Target)
	if progress {
		onDone = func(done int, t gitmonitor.Target) {
			fmt.Fprintf(os.Stderr, "\rScanning [%d/%d] %-30s", done, len(targets), t.Name)
		}
	}
	results := gitmonitor.ScanAll(gitmonitor.NewScanner(), targets, jobs, onDone)
	if progress {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 60))
	}
	return results
}

// markThirdParty flags third-party repositories and moves them behind your
// own ones, keeping the order by name within each group.
func markThirdParty(results []*gitmonitor.RepoStatus, thirdParty map[string]bool) {
	for _, r := range results {
		r.ThirdParty = thirdParty[r.Name]
	}
	sort.SliceStable(results, func(i, j int) bool { return !results[i].ThirdParty && results[j].ThirdParty })
}

// hasBehindBranch reports whether any local branch is behind its upstream.
func hasBehindBranch(s *gitmonitor.RepoStatus) bool {
	for _, b := range s.LocalBranches {
		if b.Behind > 0 {
			return true
		}
	}
	return false
}

// summarize counts repositories by state; a repository with several ahead
// branches counts once.
func summarize(results []*gitmonitor.RepoStatus) gitmonitor.Summary {
	summary := gitmonitor.Summary{Total: len(results)}
	for _, s := range results {
		if s.ErrorMsg != "" {
			summary.Errors++
			if s.Missing {
				summary.Missing++
			}
			continue
		}
		if s.ThirdParty {
			summary.ThirdParty++
			if hasBehindBranch(s) {
				summary.ThirdPartyUpdates++
			}
			continue
		}
		subAction, subPointers := s.SubmoduleAttention()
		if !s.IsClean || len(subAction) > 0 {
			summary.Dirty++
		}
		if subPointers > 0 {
			summary.SubmodulePointers++
		}
		if s.Visibility == visibility.Public {
			summary.Public++
		}
		hasAhead, hasBehind := false, false
		for _, b := range s.LocalBranches {
			hasAhead = hasAhead || b.Ahead > 0
			hasBehind = hasBehind || b.Behind > 0
		}
		if hasAhead {
			summary.Ahead++
		}
		if hasBehind {
			summary.Behind++
		}
	}
	return summary
}
