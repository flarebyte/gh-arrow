package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flarebyte/gh-arrow/internal/source"
)

func TestLoadA03(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	if _, err := Load(p, false); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p, true); err == nil {
		t.Fatal("missing explicit config accepted")
	}
	for _, text := range []string{"", "{}", "# empty\n"} {
		os.WriteFile(p, []byte(text), 0600)
		if _, err := Load(p, true); err != nil {
			t.Fatal(err)
		}
	}
	for _, text := range []string{"unknown: true", "github: [", "github: {}\ngithub: {}", "local:\n  mappings:\n    - repo: a\n      root: relative", "{}\n---\n{}", "local:\n  mappings:\n    - repo: a\n      root: /a\n    - repo: a\n      root: /b", "github:\n  mappings:\n    - repo: Owner:Repo\n      root: /a\n    - repo: owner:repo\n      root: /b"} {
		os.WriteFile(p, []byte(text), 0600)
		if _, err := Load(p, true); err == nil {
			t.Errorf("accepted %q", text)
		}
	}
}

func TestResolveMappingPrecedenceA01(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(p, []byte("github:\n  mappings:\n    - repo: Owner:Repo\n      root: "+root+"\nlocal:\n  mappings:\n    - repo: project\n      root: "+root+"\n"), 0600)
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
