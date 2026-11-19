// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
	"github.com/hmsoft0815/GitPulseMLC/pkg/visibility"
)

// ShortenPath replaces a leading prefix of path with replacement for display.
func ShortenPath(path string, prefix string, replacement string) string {
	if prefix == "" {
		return path
	}
	if strings.HasPrefix(path, prefix) {
		return strings.Replace(path, prefix, replacement, 1)
	}
	return path
}

// htmlColors maps each status part to an inline CSS colour. Mail clients such
// as Outlook drop <style> blocks, so every colour is set inline.
var htmlColors = map[partKind]string{
	kindError:      "color:#dc2626;font-weight:bold",
	kindDirty:      "color:#b45309;font-weight:bold",
	kindAhead:      "color:#2563eb;font-weight:bold",
	kindBehind:     "color:#dc2626;font-weight:bold",
	kindStash:      "color:#b45309",
	kindClean:      "color:#059669",
	kindStale:      "color:#6b7280;font-size:12px",
	kindSubmodule:  "color:#b45309;font-weight:bold",
	kindSubPointer: "color:#6b7280;font-size:12px",
	kindHint:       "color:#6b7280",
}

type htmlPart struct {
	Text  string
	Style template.CSS
}

type htmlRow struct {
	Name, Branch, Path, Error string
	HostStart                 string // heading for the first row of a remote host
	SectionStart              string // heading shown above this row, e.g. "Third party"
	Remote                    htmlPart
	Parts                     []htmlPart
}

type htmlReport struct {
	Version, Generated string
	Rows               []htmlRow
	Summary            gitmonitor.Summary
	Compact            bool
	Hidden             int
}

// HTML writes a self-contained, high-contrast report suitable for e-mail.
// In compact mode only repositories that need action are listed. With colored
// false, all text is black.
func HTML(w io.Writer, version string, results []*gitmonitor.RepoStatus, summary gitmonitor.Summary, colored bool, settings config.Settings) error {
	report := htmlReport{
		Version:   version,
		Generated: time.Now().Format("2006-01-02 15:04"),
		Summary:   summary,
		Compact:   settings.CompactMode,
	}
	thirdPartyShown, currentHost := false, ""
	for _, s := range results {
		parts, needsAction := statusParts(s)
		if settings.CompactMode && !needsAction {
			report.Hidden++
			continue
		}
		hostHeading := hostStart(s, &currentHost, &thirdPartyShown)
		row := htmlRow{
			HostStart:    hostHeading,
			SectionStart: sectionStart(s, &thirdPartyShown),
			Name:         s.Name,
			Branch:       s.CurrentBranch,
			Path:         displayPath(s, settings),
			Error:        s.ErrorMsg,
		}
		remoteCSS := "color:#6b7280;font-size:12px"
		if s.Visibility == visibility.Public && colored {
			remoteCSS = "color:#c2410c;font-weight:bold"
		}
		row.Remote = htmlPart{Text: remoteText(s), Style: template.CSS(remoteCSS)}
		for _, p := range parts {
			style := "color:#111827"
			if colored {
				style = htmlColors[p.kind]
			}
			row.Parts = append(row.Parts, htmlPart{Text: p.text, Style: template.CSS(style)})
		}
		report.Rows = append(report.Rows, row)
	}
	if err := htmlTemplate.Execute(w, report); err != nil {
		return fmt.Errorf("render html report: %w", err)
	}
	return nil
}

// displayPath shortens local paths; remote paths stay as the host reports them.
func displayPath(s *gitmonitor.RepoStatus, settings config.Settings) string {
	if s.Host != "" {
		return s.Path
	}
	return ShortenPath(s.Path, settings.ReplacePathPrefix, settings.ReplaceWith)
}

// sectionStart returns the heading for the first third-party row.
func sectionStart(s *gitmonitor.RepoStatus, shown *bool) string {
	if !s.ThirdParty || *shown {
		return ""
	}
	*shown = true
	return "Third party"
}

// hostStart returns the heading for the first row of a remote host and
// resets the third-party heading for it.
func hostStart(s *gitmonitor.RepoStatus, host *string, thirdPartyShown *bool) string {
	if s.Host == *host {
		return ""
	}
	*host, *thirdPartyShown = s.Host, false
	return "Host " + s.Host
}

var htmlTemplate = template.Must(template.New("report").Parse(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>GitPulseMLC Report</title></head>
<body style="font-family:Arial,Helvetica,sans-serif;font-size:14px;color:#111827;background:#ffffff;margin:0;padding:20px">
<h2 style="margin:0 0 4px 0">GitPulseMLC Report</h2>
<p style="margin:0 0 16px 0;color:#6b7280;font-size:12px">Generated {{.Generated}}{{if .Compact}} · showing only repositories that need attention ({{.Hidden}} clean hidden){{end}}</p>
{{if .Rows}}<table cellpadding="0" cellspacing="0" style="border-collapse:collapse;width:100%;margin-bottom:20px">
<thead><tr>
<th style="text-align:left;padding:10px;background:#1f2937;color:#ffffff;font-size:12px;text-transform:uppercase">Project</th>
<th style="text-align:left;padding:10px;background:#1f2937;color:#ffffff;font-size:12px;text-transform:uppercase">Branch</th>
<th style="text-align:left;padding:10px;background:#1f2937;color:#ffffff;font-size:12px;text-transform:uppercase">Remote</th>
<th style="text-align:left;padding:10px;background:#1f2937;color:#ffffff;font-size:12px;text-transform:uppercase">Status</th>
<th style="text-align:left;padding:10px;background:#1f2937;color:#ffffff;font-size:12px;text-transform:uppercase">Path</th>
</tr></thead>
<tbody>
{{range .Rows}}{{if .HostStart}}<tr><td colspan="5" style="padding:20px 10px 6px 10px;border-bottom:2px solid #1f2937;font-weight:bold;font-size:15px">{{.HostStart}} <span style="font-weight:normal;color:#6b7280;font-size:12px">remote repositories, scanned there</span></td></tr>
{{end}}{{if .SectionStart}}<tr><td colspan="5" style="padding:16px 10px 6px 10px;border-bottom:2px solid #d1d5db;font-weight:bold">{{.SectionStart}} <span style="font-weight:normal;color:#6b7280;font-size:12px">checked out to look at or test; local changes are only hints</span></td></tr>
{{end}}<tr>
<td style="padding:10px;border-bottom:1px solid #e5e7eb"><strong>{{.Name}}</strong></td>
<td style="padding:10px;border-bottom:1px solid #e5e7eb;font-family:Consolas,Menlo,monospace">{{.Branch}}</td>
<td style="padding:10px;border-bottom:1px solid #e5e7eb"><span style="{{.Remote.Style}}">{{.Remote.Text}}</span></td>
<td style="padding:10px;border-bottom:1px solid #e5e7eb">{{range $i, $p := .Parts}}{{if $i}} {{end}}<span style="{{$p.Style}}">{{$p.Text}}</span>{{end}}{{if .Error}}<br><span style="color:#6b7280;font-size:12px">{{.Error}}</span>{{end}}</td>
<td style="padding:10px;border-bottom:1px solid #e5e7eb;color:#6b7280;font-size:12px">{{.Path}}</td>
</tr>
{{end}}</tbody></table>
{{else}}<p style="padding:16px;background:#ecfdf5;border:1px solid #a7f3d0;color:#065f46">Nothing to do — all repositories are clean and in sync.</p>
{{end}}<table cellpadding="0" cellspacing="0" style="border:1px solid #d1d5db;background:#f9fafb"><tr><td style="padding:16px">
<strong style="display:block;font-size:16px;margin-bottom:8px">Summary</strong>
Total <strong>{{.Summary.Total}}</strong> &nbsp;·&nbsp;
Dirty <strong style="color:#b45309">{{.Summary.Dirty}}</strong> &nbsp;·&nbsp;
Ahead <strong style="color:#2563eb">{{.Summary.Ahead}}</strong> &nbsp;·&nbsp;
Behind <strong style="color:#dc2626">{{.Summary.Behind}}</strong> &nbsp;·&nbsp;
Errors <strong style="color:#dc2626">{{.Summary.Errors}}</strong>{{if .Summary.ThirdParty}} &nbsp;·&nbsp;
<span style="color:#6b7280">Third party {{.Summary.ThirdParty}} ({{.Summary.ThirdPartyUpdates}} with upstream updates)</span>{{end}}{{if .Summary.Public}} &nbsp;·&nbsp;
Public <strong style="color:#c2410c">{{.Summary.Public}}</strong>{{end}}{{if .Summary.SubmodulePointers}} &nbsp;·&nbsp;
<span style="color:#6b7280">Submodule pointers to commit {{.Summary.SubmodulePointers}}</span>{{end}}
</td></tr></table>
<p style="margin-top:30px;padding-top:10px;border-top:1px solid #f3f4f6;font-size:11px;color:#9ca3af">Generated by GitPulseMLC {{.Version}} · https://mlcgo.eu/products/gitpulse/</p>
</body></html>
`))
