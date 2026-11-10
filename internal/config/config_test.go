package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flarebyte/gh-arrow/internal/source"
)

func TestLoadA03(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.cue")
	if _, err := Load(p, false); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p, true); err == nil {
		t.Fatal("missing explicit config accepted")
	}
	for _, text := range []string{"", "{}", "// empty\n"} {
		os.WriteFile(p, []byte(text), 0600)
		if _, err := Load(p, true); err != nil {
			t.Fatal(err)
		}
	}
	for _, text := range []string{`unknown: true`, `github: [`, `local: mappings: [{repo: "a", root: "relative"}]`, `local: mappings: [{repo: "a", root: "/a"}, {repo: "a", root: "/b"}]`, `github: mappings: [{repo: "Owner:Repo", root: "/a"}, {repo: "owner:repo", root: "/b"}]`, `local: mappings: [{repo: "a"}]`, `local: mappings: [{repo: string, root: "/a"}]`, `import "os"`, `import "strings"; local: mappings: []`, `local: mappings: [{repo: "a", root: "/a", extra: true}]`} {
		os.WriteFile(p, []byte(text), 0600)
		if _, err := Load(p, true); err == nil {
			t.Errorf("accepted %q", text)
		}
	}
}

func TestMalformedConfigDoesNotEchoScalarValues(t *testing.T) {
	name := filepath.Join(t.TempDir(), "config.cue")
	const secret = "token42"
	for _, data := range []string{`github: "` + secret + `"`, `local: mappings: "` + secret + `"`} {
		if err := os.WriteFile(name, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := Load(name, true)
		if err == nil || strings.Contains(err.Error(), secret) {
			t.Fatalf("parser diagnostic leaked input: %v", err)
		}
	}
}

func TestResolveMappingPrecedenceA01(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(t.TempDir(), "config.cue")
	os.WriteFile(p, []byte(fmt.Sprintf(`github: mappings: [{repo: "Owner:Repo", root: %q}]
local: mappings: [{repo: "project", root: %q}]`, root, root)), 0600)
	c, err := Load(p, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"github:owner:repo:file", "local:project:file"} {
		s, _ := source.Parse(raw)
		got, err := c.Root(s)
		if err != nil || got != root {
			t.Fatalf("%s: %q %v", raw, got, err)
		}
	}
	s, _ := source.Parse("github:other:repo:file")
	if got, err := c.Root(s); err != nil || got != "" {
		t.Fatal(got, err)
	}
	s, _ = source.Parse("local:unknown:file")
	if _, err := c.Root(s); err == nil {
		t.Fatal("unknown alias accepted")
	}
}

func TestDefaultHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(t.TempDir())
	os.WriteFile("config.yaml", []byte("invalid"), 0600)
	if _, err := Load("", false); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(home, ".gh-arrow"), []byte(`local: mappings: [{repo: "home", root: `+fmt.Sprintf("%q", home)+`}]`), 0600)
	c, err := Load("", false)
	if err != nil || len(c.Local.Mappings) != 1 {
		t.Fatalf("home config: %+v %v", c, err)
	}
}

func TestExplicitOverrideAndLimits(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.WriteFile(filepath.Join(home, ".gh-arrow"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(t.TempDir(), "config.cue")
	if err := os.WriteFile(name, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(name, true); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(strings.Repeat(" ", 1024*1024+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(name, true); err == nil {
		t.Fatal("oversize config accepted")
	}
	missing := filepath.Join(home, "private-name")
	if _, err := Load(missing, true); err == nil || strings.Contains(err.Error(), home) || strings.Contains(err.Error(), "private-name") {
		t.Fatalf("unsafe I/O diagnostic: %v", err)
	}
}
