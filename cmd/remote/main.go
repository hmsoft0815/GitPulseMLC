// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	remoteHost := flag.String("host", "", "Remote host (e.g. user@host)")
	configFile := flag.String("config", "", "Local path to the configuration file")
	arch := flag.String("arch", "", "Target architecture (linux-amd64, darwin-arm64, windows-amd64, etc.)")
	flag.Parse()

	if *remoteHost == "" || *configFile == "" {
		fmt.Println("Usage: remote_gitpulse -host user@host -config path/to/repos.ini [-arch arch]")
		flag.Usage()
		os.Exit(1)
	}

	// 1. Determine the binary to use
	targetArch := *arch
	if targetArch == "" {
		// Default to current arch if not specified
		targetArch = fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	}

	binaryName := "gitpulse-" + targetArch
	if strings.Contains(targetArch, "windows") {
		binaryName += ".exe"
	}

	localBinaryPath := filepath.Join("bin", binaryName)
	if _, err := os.Stat(localBinaryPath); os.IsNotExist(err) {
		log.Fatalf("Binary not found: %s. Please run 'task build-cross' first.", localBinaryPath)
	}

	// 2. Prepare remote execution
	remoteTempDir := fmt.Sprintf("/tmp/gitpulse-%d", os.Getpid())
	remoteBinaryPath := filepath.Join(remoteTempDir, "gitpulse")
	remoteConfigPath := filepath.Join(remoteTempDir, "repos.ini")

	fmt.Printf("🚀 Preparing remote execution on %s...\n", *remoteHost)

	// Create temp dir on remote
	runSSH(*remoteHost, fmt.Sprintf("mkdir -p %s", remoteTempDir))

	// SCP binary and config
	copyToRemote(*remoteHost, localBinaryPath, remoteBinaryPath)
	copyToRemote(*remoteHost, *configFile, remoteConfigPath)

	// Make binary executable
	runSSH(*remoteHost, fmt.Sprintf("chmod +x %s", remoteBinaryPath))

	// 3. Execute
	fmt.Println("📡 Executing GitPulseMLC on remote host...")
	fmt.Println(strings.Repeat("─", 40))
	
	// Pass through any additional arguments to the remote binary
	// (Note: this is simplified, might need better escaping)
	cmdArgs := strings.Join(flag.Args(), " ")
	runSSH(*remoteHost, fmt.Sprintf("%s --config %s %s", remoteBinaryPath, remoteConfigPath, cmdArgs))

	// 4. Cleanup
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println("🧹 Cleaning up...")
	runSSH(*remoteHost, fmt.Sprintf("rm -rf %s", remoteTempDir))
	fmt.Println("✅ Done.")
}

func runSSH(host, command string) {
	cmd := exec.Command("ssh", host, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("SSH command failed: %v", err)
	}
}

func copyToRemote(host, src, dst string) {
	cmd := exec.Command("scp", src, fmt.Sprintf("%s:%s", host, dst))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("SCP failed: %v", err)
	}
}
