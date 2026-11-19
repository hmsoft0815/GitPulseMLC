// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package visibility

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"os/exec"
	"path"
	"strings"
	"time"
)

// localSuffixes are domain suffixes that only resolve inside a local network.
var localSuffixes = []string{".local", ".lan", ".home", ".internal", ".intranet", ".localdomain", ".home.arpa", ".fritz.box"}

// IsLocalHost reports whether host lives on the local network: no dot in the
// name, a local-only suffix, a loopback, private or link-local IP, or a match
// for one of the glob patterns in privateHosts (e.g. "git.example.com",
// "*.corp.example").
func IsLocalHost(host string, privateHosts []string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" {
		return true
	}
	for _, pattern := range privateHosts {
		if ok, _ := path.Match(strings.ToLower(pattern), host); ok {
			return true
		}
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
	}
	if !strings.Contains(host, ".") {
		return true
	}
	for _, suffix := range localSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

// SSHHostname resolves an SSH alias such as "github-work" to the host name
// ssh would connect to, using "ssh -G". It runs offline and returns host
// unchanged if ssh is missing or knows no other name.
func SSHHostname(host string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// "--" keeps a host that starts with "-" from being read as an option.
	out, err := exec.CommandContext(ctx, "ssh", "-G", "--", host).Output()
	if err != nil {
		return host
	}
	return parseSSHConfig(out, host)
}

// parseSSHConfig picks the "hostname" line from "ssh -G" output.
func parseSSHConfig(out []byte, fallback string) string {
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		key, value, ok := strings.Cut(sc.Text(), " ")
		if ok && strings.EqualFold(key, "hostname") && value != "" {
			return strings.ToLower(value)
		}
	}
	return fallback
}
