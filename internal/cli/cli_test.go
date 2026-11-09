package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flarebyte/gh-arrow/internal/github"
)

func localFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "docs"), 0700)
	os.WriteFile(filepath.Join(root, "docs/a.md"), []byte{0, 255, '\r', '\n'}, 0600)
	os.WriteFile(filepath.Join(root, "docs/z.md"), []byte("z"), 0600)
	config := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(config, []byte("local:\n  mappings:\n    - repo: fixture\n      root: "+root+"\ngithub:\n  mappings:\n    - repo: o:r\n      root: "+root+"\n"), 0600)
	return root, config
}
func TestLocalCommandsAndNoFallbackA01A02A13(t *testing.T) {
	root, config := localFixture(t)
	calls := 0
	r := Runner{NewRemote: func(context.Context) (*github.Client, error) {
		calls++
		return nil, errors.New("network must not be used")
	}}
	cases := []struct {
		args []string
		exit int
		out  string
		code string
	}{
		{[]string{"cat", "local:fixture:docs/a.md"}, 0, string([]byte{0, 255, '\r', '\n'}), ""},
		{[]string{"cat", "github:o:r:missing"}, 1, "", "ARROW_NOT_FOUND"},
		{[]string{"ls", "local:fixture:docs/"}, 0, "a.md\nz.md\n", ""},
		{[]string{"ls", "local:fixture:docs/*.md", "--json"}, 0, "[{\"path\":\"a.md\"},{\"path\":\"z.md\"}]\n", ""},
		{[]string{"ls", "local:fixture:docs/none*", "--json"}, 0, "[]\n", ""},
		{[]string{"ls", "local:fixture:docs/a.md/*", "--json"}, 0, "[]\n", ""},
		{[]string{"ls", "local:fixture:docs/missing"}, 1, "", "ARROW_NOT_FOUND"},
		{[]string{"ls", "local:fixture:docs/a.md"}, 0, "a.md\n", ""},
		{[]string{"cp", "local:fixture:docs/a.md", filepath.Join(root, "copied")}, 0, "", ""},
		{[]string{"cp", "local:fixture:docs/a.md", filepath.Join(root, "copied")}, 1, "", "ARROW_EXISTS"},
		{[]string{"cp", "local:fixture:docs/a.md", filepath.Join(root, "copied"), "--force"}, 0, "", ""},
		{[]string{"cat", "local:fixture:docs/a.md", "--ref", "main"}, 2, "", "ARROW_USAGE"},
	}
	for _, tt := range cases {
		var out, err bytes.Buffer
		args := append([]string{"--config", config}, tt.args...)
		exit := r.Run(context.Background(), args, &out, &err)
		if exit != tt.exit || out.String() != tt.out || tt.code != "" && !strings.HasPrefix(err.String(), tt.code+":") {
			t.Errorf("%v: exit=%d out=%q err=%q", args, exit, out.String(), err.String())
		}
	}
	if calls != 0 {
		t.Fatal("mapped sources called remote", calls)
	}
}

func TestUsageAndHelpOfflineA04(t *testing.T) {
	_, config := localFixture(t)
	r := Runner{NewRemote: func(context.Context) (*github.Client, error) { t.Fatal("unexpected remote call"); return nil, nil }}
	for _, args := range [][]string{{}, {"unknown"}, {"cat", "local:a:x", "--json"}, {"cp", "local:a:x", "out", "--org", "owner"}, {"ls", "--org", "owner", "*.md"}, {"ls", "--org", "owner", "README", "--ref", "main"}, {"ls", "local:a:x", "--topic", "cli"}, {"cat", "local:a:"}, {"cat", "local:a:../x"}, {"ls", "local:a:a?"}} {
		var out, err bytes.Buffer
		if code := r.Run(context.Background(), append([]string{"--config", config}, args...), &out, &err); code != 2 || out.Len() != 0 || !strings.HasPrefix(err.String(), "ARROW_USAGE:") {
			t.Errorf("%v: %d %q %q", args, code, out.String(), err.String())
		}
	}
	for _, cmd := range []string{"", "cp", "cat", "ls"} {
		var out, err bytes.Buffer
		args := []string{"--config", "/does/not/exist"}
		if cmd != "" {
			args = append(args, cmd)
		}
		args = append(args, "--help")
		if code := r.Run(context.Background(), args, &out, &err); code != 0 || out.Len() == 0 || err.Len() != 0 {
			t.Errorf("help: %d %q %q", code, out.String(), err.String())
		}
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }
func TestOutputFailure(t *testing.T) {
	_, config := localFixture(t)
	var err bytes.Buffer
	if code := new(Runner).Run(context.Background(), []string{"cat", "local:fixture:docs/a.md", "--config", config}, brokenWriter{}, &err); code != 1 {
		t.Fatal(code, err.String())
	}
}

func TestEscapedFilenamesA18(t *testing.T) {
	root, config := localFixture(t)
	if err := os.WriteFile(filepath.Join(root, "docs/line\nbreak.md"), []byte("text"), 0600); err != nil {
		t.Skipf("control-character filename unavailable: %v", err)
	}
	var out, err bytes.Buffer
	if code := new(Runner).Run(context.Background(), []string{"ls", "local:fixture:docs/", "--config", config}, &out, &err); code != 0 || !strings.Contains(out.String(), "line\\nbreak.md\n") {
		t.Fatal(code, out.String(), err.String())
	}
	out.Reset()
	err.Reset()
	if code := new(Runner).Run(context.Background(), []string{"ls", "local:fixture:docs/", "--config", config, "--json"}, &out, &err); code != 0 || !strings.Contains(out.String(), `line\nbreak.md`) {
		t.Fatal(code, out.String(), err.String())
	}
}

func TestVersionOfflineA19(t *testing.T) {
	r := Runner{NewRemote: func(context.Context) (*github.Client, error) {
		t.Fatal("version accessed remote credentials")
		return nil, nil
	}}
	var out, err bytes.Buffer
	if code := r.Run(context.Background(), []string{"--version", "--config", "/missing/config"}, &out, &err); code != 0 || !strings.HasPrefix(out.String(), "gh-arrow ") || strings.Contains(out.String(), "unknown") || err.Len() != 0 {
		t.Fatal(code, out.String(), err.String())
	}
}

func TestCatTreatsFilenameMetacharactersLiterally(t *testing.T) {
	root, config := localFixture(t)
	if err := os.WriteFile(filepath.Join(root, "notes[1].md"), []byte("literal bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, err bytes.Buffer
	if code := new(Runner).Run(context.Background(), []string{"cat", "local:fixture:notes[1].md", "--config", config}, &out, &err); code != 0 || out.String() != "literal bytes" {
		t.Fatal(code, out.String(), err.String())
	}
}
