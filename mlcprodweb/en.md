# Keep Your Git Projects Under Control

**The Terminal Heartbeat for Developers**
GitPulseMLC is the ultimate companion for developers managing dozens of local repositories. Instead of manually checking each folder, get a bird's-eye view of your entire development environment in a split second.

**Insights Without the Overhead**
Stop guessing which project has unpushed commits or forgotten stashes. GitPulseMLC provides detailed summaries of your worktree state, including specific file types changed and branch staleness. It's built for speed, using Go's concurrency to handle massive project lists without breaking a sweat.

**Designed for Automation**
Whether you need a quick status check in your terminal or an automated HTML report for your team, GitPulseMLC has you covered. With JSON export and high-contrast HTML output, it integrates perfectly into your existing workflows and monitoring scripts.

**Quickstart**
Get GitPulseMLC running in seconds using Go. Build the TUI client and run a compact scan to see only the repositories that need your attention:

```bash
# Build the binary
go build -o gitpulse ./cmd/tui

# Run your first scan in compact mode
./gitpulse --compact
```
