// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/render"
	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
)

// gitIn runs git in dir with a fixed identity and fails the test on error.
func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	base := []string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always", "-c", "init.defaultBranch=main"}
	cmd := exec.Command("git", append(base, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// commitFile writes a file and commits it.
func commitFile(t *testing.T, dir, name, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(msg+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", name)
	gitIn(t, dir, "commit", "-q", "-m", msg)
}

// fixtureProjects builds repositories covering every part of the JSON
// output: a diverged clone with a submodule, a third-party clone, a dirty
// repository without remote, a missing path and a folder without repository.
func fixtureProjects(t *testing.T) (*config.Config, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	mk := func(name string) string {
		dir := filepath.Join(root, name)
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		return dir
	}

	upstream := mk("upstream.git")
	gitIn(t, upstream, "init", "-q", "--bare")
	seed := mk("seed")
	gitIn(t, seed, "init", "-q")
	commitFile(t, seed, "a.txt", "first")
	gitIn(t, seed, "push", "-q", upstream, "main")

	docs := mk("docs")
	gitIn(t, docs, "init", "-q")
	commitFile(t, docs, "README.md", "docs")

	gitIn(t, root, "clone", "-q", upstream, "app")
	app := filepath.Join(root, "app")
	gitIn(t, app, "submodule", "add", "-q", docs, ".mlcai")
	gitIn(t, app, "commit", "-q", "-m", "add submodule")
	commitFile(t, app, "local.txt", "local change")
	commitFile(t, seed, "b.txt", "upstream change")
	gitIn(t, seed, "push", "-q", upstream, "main")
	gitIn(t, app, "fetch", "-q")
	if err := os.WriteFile(filepath.Join(app, "dirty.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	gitIn(t, root, "clone", "-q", upstream, "vendor")
	solo := mk("solo")
	gitIn(t, solo, "init", "-q")
	commitFile(t, solo, "x.txt", "solo")
	empty := mk("empty")

	cfg := &config.Config{
		Projects: map[string]string{
			"app": app, "vendor": filepath.Join(root, "vendor"), "solo": solo,
			"gone": filepath.Join(root, "gone"), "empty": empty,
		},
		ThirdParty: map[string]bool{"vendor": true},
	}
	return cfg, root
}

// renderFixture scans the fixture like `gitpulse --json --offline` does.
func renderFixture(t *testing.T) map[string]any {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir()) // never read the user's visibility cache
	cfg, _ := fixtureProjects(t)
	results := scanProjects(cfg.Projects, 2, false)
	markThirdParty(results, cfg.ThirdParty)
	checkVisibility(results, cfg.Settings, true)

	var buf bytes.Buffer
	if err := render.JSON(&buf, "test", results, summarize(results)); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("output is no JSON: %v\n%s", err, buf.String())
	}
	return doc
}

func TestJSONMatchesSchema(t *testing.T) {
	doc := renderFixture(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	v := validator{defs: schema["definitions"].(map[string]any)}
	v.check("$", doc, schema)
	for _, e := range v.errs {
		t.Error(e)
	}
}

func TestJSONRepositoryFields(t *testing.T) {
	doc := renderFixture(t)
	repos := map[string]map[string]any{}
	for _, r := range doc["results"].([]any) {
		repo := r.(map[string]any)
		repos[repo["name"].(string)] = repo
	}
	app := repos["app"]
	want := map[string]any{
		"ahead": 2.0, "behind": 1.0, "is_tracking": true, "is_diverged": true,
		"has_remote_updates": true, "remote_branch_name": "origin/main", "is_clean": false,
	}
	for k, w := range want {
		if app[k] != w {
			t.Errorf("app.%s = %v, want %v", k, app[k], w)
		}
	}
	if got := app["incoming_commits"].([]any); len(got) != 1 || got[0] != "upstream change" {
		t.Errorf("app.incoming_commits = %v, want [upstream change]", got)
	}
	if subs := app["submodules"].([]any); len(subs) != 1 {
		t.Errorf("app.submodules = %v, want one", subs)
	}
	if repos["vendor"]["third_party"] != true || repos["gone"]["missing"] != true || repos["empty"]["missing"] != true {
		t.Errorf("third_party/missing flags wrong: vendor=%v gone=%v empty=%v",
			repos["vendor"]["third_party"], repos["gone"]["missing"], repos["empty"]["missing"])
	}
	summary := doc["summary"].(map[string]any)
	if summary["third_party"] != 1.0 || summary["missing"] != 2.0 || summary["errors"] != 2.0 {
		t.Errorf("summary = %v", summary)
	}
}

// validator checks a JSON document against the subset of JSON Schema that
// docs/schema.json uses: $ref, type, properties, required, items and enum.
// Unlike JSON Schema it also rejects properties the schema does not list,
// so undocumented output fails the test.
type validator struct {
	defs map[string]any
	errs []string
}

func (v *validator) fail(path, format string, args ...any) {
	v.errs = append(v.errs, path+": "+fmt.Sprintf(format, args...))
}

func (v *validator) check(path string, value any, schema map[string]any) {
	if ref, ok := schema["$ref"].(string); ok {
		schema = v.defs[strings.TrimPrefix(ref, "#/definitions/")].(map[string]any)
	}
	if t, ok := schema["type"]; ok && !typeMatches(t, value) {
		v.fail(path, "type %s, want %v", jsonType(value), t)
		return
	}
	if enum, ok := schema["enum"].([]any); ok && !slices.Contains(enum, value) {
		v.fail(path, "%v not in %v", value, enum)
	}
	switch val := value.(type) {
	case map[string]any:
		v.checkObject(path, val, schema)
	case []any:
		if items, ok := schema["items"].(map[string]any); ok {
			for i, item := range val {
				v.check(fmt.Sprintf("%s[%d]", path, i), item, items)
			}
		}
	}
}

func (v *validator) checkObject(path string, obj map[string]any, schema map[string]any) {
	props, _ := schema["properties"].(map[string]any)
	for _, req := range asSlice(schema["required"]) {
		if _, ok := obj[req.(string)]; !ok {
			v.fail(path, "required field %q missing", req)
		}
	}
	if props == nil {
		return // free-form object, e.g. changed_files
	}
	for key, val := range obj {
		sub, ok := props[key].(map[string]any)
		if !ok {
			v.fail(path, "field %q is not in docs/schema.json", key)
			continue
		}
		v.check(path+"."+key, val, sub)
	}
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func typeMatches(want any, value any) bool {
	got := jsonType(value)
	for _, t := range append(asSlice(want), want) {
		if t == got || (t == "number" && got == "integer") {
			return true
		}
	}
	return false
}

func jsonType(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		if v == float64(int64(v)) {
			return "integer"
		}
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	default:
		return "object"
	}
}
