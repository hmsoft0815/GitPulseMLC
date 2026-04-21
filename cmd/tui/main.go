// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gitpulsemcl/internal/ui/render"
	"gitpulsemcl/pkg/config"
	"gitpulsemcl/pkg/gitmonitor"

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

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "GitPulseMLC  - Monitor the heartbeat of your local Git repositories\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  gitpulse [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConfig Lookup Order:\n")
		fmt.Fprintf(os.Stderr, "  1. --config flag\n")
		fmt.Fprintf(os.Stderr, "  2. GITPULSE_CONFIG environment variable\n")
		fmt.Fprintf(os.Stderr, "  3. config/repos.ini (current directory)\n")
		fmt.Fprintf(os.Stderr, "  4. ~/.gitpulsemlc/repos.ini (user home)\n")
	}

	flag.Parse()

	if *printVersion {
		fmt.Printf("GitPulseMLC version %s\n", version)
		os.Exit(0)
	}

	if *noColor {
		lipgloss.SetColorProfile(termenv.Ascii)
	}

	details := *showDetails || *showAll

	configPath := findConfig(*configFlag)
	if configPath == "" {
		log.Fatalf("Configuration file not found. Please create config/repos.ini or ~/.gitpulsemlc/repos.ini")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("failed to load config from %s: %v", configPath, err)
	}

	if *compactFlag {
		cfg.Settings.CompactMode = true
	}

	scanner := gitmonitor.NewScanner()

	// 1. Scan all
	var allResults []*gitmonitor.RepoStatus
	projectNames := make([]string, 0, len(cfg.Projects))
	for name := range cfg.Projects {
		projectNames = append(projectNames, name)
	}
	sort.Strings(projectNames)

	isReportMode := *jsonOutput || *htmlOutput

	for i, name := range projectNames {
		if *showProgress && !isReportMode {
			fmt.Fprintf(os.Stderr, "\rScanning [%d/%d] %-30s", i+1, len(projectNames), name)
		}
		path := cfg.Projects[name]
		status, _ := scanner.Scan(name, path)
		allResults = append(allResults, status)
	}
	if *showProgress && !isReportMode {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 60))
	}

	summary := gitmonitor.Summary{Total: len(allResults)}
	for _, s := range allResults {
		if s.ErrorMsg != "" {
			summary.Errors++
			continue
		}
		if !s.IsClean {
			summary.Dirty++
		}
		hasAhead, hasBehind := false, false
		for _, b := range s.LocalBranches {
			if b.Ahead > 0 {
				hasAhead = true
			}
			if b.Behind > 0 {
				hasBehind = true
			}
		}
		if hasAhead {
			summary.Ahead++
		}
		if hasBehind {
			summary.Behind++
		}
	}

	// 2. Output processing
	if *jsonOutput {
		render.JSON(version, allResults, summary)
		return
	}

	if *htmlOutput {
		render.HTML(version, allResults, summary, !*noColor, cfg.Settings)
		return
	}

	render.TUI(allResults, summary, cfg, details)
}
