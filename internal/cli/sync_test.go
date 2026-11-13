package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flarebyte/gh-arrow/internal/github"
)

func syncFixture(t *testing.T, entries string) (string, string, string, Runner) {
	t.Helper()
	upstream, config := localFixture(t)
	project := t.TempDir()
	manifest := filepath.Join(project, "sync.gh-arrow.cue")
	if err := os.WriteFile(manifest, []byte("version: 1\nfiles: ["+entries+"]"), 0600); err != nil {
		t.Fatal(err)
	}
	r := Runner{NewRemote: func(context.Context) (*github.Client, error) {
		t.Fatal("mapped sync must not call remote")
		return nil, nil
	}}
	return upstream, project, config, r
}

func TestSyncLifecycleA22(t *testing.T) {
	_, project, config, r := syncFixture(t, `{source: "github:o:r:docs/z.md", destination: "z.md"}, {source: "github:o:r:docs/a.md", destination: "a.bin"}`)
	manifest := filepath.Join(project, "sync.gh-arrow.cue")
	run := func(args ...string) (int, string, string) {
		var out, stderr bytes.Buffer
		code := r.Run(context.Background(), append([]string{"--config", config, "sync", "--file", manifest}, args...), &out, &stderr)
		return code, out.String(), stderr.String()
	}
	for _, args := range [][]string{{"status"}, {"--dry-run"}, {"check"}} {
		code, out, stderr := run(args...)
		expected := 0
		if args[0] == "check" {
			expected = 3
		}
		if code != expected || !strings.Contains(out, "missing\ta.bin") || !strings.Contains(out, "missing\tz.md") {
			t.Fatalf("%v: %d %q %q", args, code, out, stderr)
		}
		if _, err := os.Stat(filepath.Join(project, "a.bin")); !os.IsNotExist(err) {
			t.Fatal("preview wrote destination")
		}
	}
	if code, out, stderr := run(); code != 0 {
		t.Fatalf("apply: %d %q %q", code, out, stderr)
	}
	data, err := os.ReadFile(filepath.Join(project, "a.bin"))
	if err != nil || !bytes.Equal(data, []byte{0, 255, '\r', '\n'}) {
		t.Fatalf("raw bytes: %q %v", data, err)
	}
	if code, out, stderr := run("check", "--json"); code != 0 || strings.Count(out, `"status":"unchanged"`) != 2 {
		t.Fatalf("%d %q %q", code, out, stderr)
	}
	if err := os.WriteFile(filepath.Join(project, "z.md"), []byte("old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if code, out, stderr := run("diff"); code != 0 || !strings.Contains(out, "-old\n") || !strings.Contains(out, "+z\n\\ No newline at end of file") {
		t.Fatalf("diff: %d %q %q", code, out, stderr)
	}
	if code, _, stderr := run(); code != 0 {
		t.Fatal(code, stderr)
	}
	if code, _, stderr := run("check"); code != 0 {
		t.Fatal(code, stderr)
	}
}

func TestSyncFetchFailurePreservesAllA23(t *testing.T) {
	_, project, config, r := syncFixture(t, `{source:"github:o:r:docs/z.md",destination:"a"},{source:"github:o:r:missing",destination:"z"}`)
	if err := os.WriteFile(filepath.Join(project, "a"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := r.Run(context.Background(), []string{"sync", "--file", filepath.Join(project, "sync.gh-arrow.cue"), "--config", config}, &out, &stderr)
	data, err := os.ReadFile(filepath.Join(project, "a"))
	if code != 1 || out.Len() != 0 || string(data) != "original" || err != nil {
		t.Fatal(code, out.String(), stderr.String(), string(data), err)
	}
}

func TestSyncInvalidManifestA24(t *testing.T) {
	for _, body := range []string{
		`version:1,files:[{source:"github:o:r:docs/z.md",destination:"../outside"}]`,
		`version:1,files:[{source:"github:o:r:docs/z.md",destination:"/outside"}]`,
		`version:1,files:[{source:"github:o:r:docs/z.md",destination:"x"},{source:"github:o:r:docs/a.md",destination:"./x"}]`,
		`version:1,files:[{source:"github:o:r:docs/z.md",destination:"x",ref:"main"}]`,
		`version:1,files:[{source:"github:o:r:docs/*.md",destination:"x"}]`,
		`version:1,files:[{source:"github:o:r:",destination:"x"}]`,
		`version:1,files:[{source:"github:o:r:docs/z.md",destination:"sync.gh-arrow.cue"}]`,
		`import "os"`, `version:2,files:[]`,
	} {
		t.Run(fmt.Sprintf("case-%d", len(body)), func(t *testing.T) {
			_, project, config, r := syncFixture(t, "")
			manifest := filepath.Join(project, "sync.gh-arrow.cue")
			if err := os.WriteFile(manifest, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			if code := r.Run(context.Background(), []string{"sync", "--file", manifest, "--config", config}, &out, &stderr); code != 2 || out.Len() != 0 {
				t.Fatal(code, out.String(), stderr.String())
			}
		})
	}
}

func TestSyncRejectsNonFilesA25(t *testing.T) {
	for _, kind := range []string{"directory source", "symlink destination", "symlink parent", "missing parent"} {
		t.Run(kind, func(t *testing.T) {
			_, project, config, r := syncFixture(t, `{source:"github:o:r:docs/z.md",destination:"target"}`)
			manifest := filepath.Join(project, "sync.gh-arrow.cue")
			switch kind {
			case "directory source":
				os.WriteFile(manifest, []byte(`version:1,files:[{source:"github:o:r:docs",destination:"target"}]`), 0600)
			case "symlink destination":
				if err := os.Symlink(filepath.Join(project, "sync.gh-arrow.cue"), filepath.Join(project, "target")); err != nil {
					t.Skip(err)
				}
			case "symlink parent":
				if err := os.Symlink(t.TempDir(), filepath.Join(project, "link")); err != nil {
					t.Skip(err)
				}
				os.WriteFile(manifest, []byte(`version:1,files:[{source:"github:o:r:docs/z.md",destination:"link/target"}]`), 0600)
			case "missing parent":
				os.WriteFile(manifest, []byte(`version:1,files:[{source:"github:o:r:docs/z.md",destination:"missing/target"}]`), 0600)
			}
			var out, stderr bytes.Buffer
			if code := r.Run(context.Background(), []string{"sync", "--file", manifest, "--config", config}, &out, &stderr); code != 1 {
				t.Fatal(code, out.String(), stderr.String())
			}
		})
	}
}

func TestSyncFlags(t *testing.T) {
	for _, args := range [][]string{{"sync", "--ref", "main"}, {"sync", "--force"}, {"sync", "diff", "--json"}, {"sync", "status", "--dry-run"}, {"sync", "other"}, {"cat", "local:x:y", "--file", "x"}, {"--version", "--dry-run"}} {
		var out, stderr bytes.Buffer
		if code := (Runner{}).Run(context.Background(), args, &out, &stderr); code != 2 {
			t.Fatal(args, code, out.String(), stderr.String())
		}
	}
	var out, stderr bytes.Buffer
	if code := (Runner{}).Run(context.Background(), []string{"sync", "--help", "--file", "/missing"}, &out, &stderr); code != 0 || !strings.Contains(out.String(), "sync") {
		t.Fatal(code, out.String(), stderr.String())
	}
}

func TestSyncDefaultManifestEmptyAndBinaryDiff(t *testing.T) {
	_, project, config, r := syncFixture(t, "")
	t.Chdir(project)
	var out, stderr bytes.Buffer
	if code := r.Run(context.Background(), []string{"sync", "check", "--config", config, "--json"}, &out, &stderr); code != 0 || out.String() != "[]\n" {
		t.Fatal(code, out.String(), stderr.String())
	}
	if err := os.WriteFile("sync.gh-arrow.cue", []byte(`version:1,files:[{source:"github:o:r:docs/a.md",destination:"binary"}]`), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	if code := r.Run(context.Background(), []string{"sync", "diff", "--config", config}, &out, &stderr); code != 0 || !strings.Contains(out.String(), "Binary files /dev/null and b/binary differ") {
		t.Fatal(code, out.String(), stderr.String())
	}
	if _, err := os.Stat("binary"); !os.IsNotExist(err) {
		t.Fatal("diff wrote missing file")
	}
}

func TestSyncDiffMissingEmptyFile(t *testing.T) {
	upstream, project, config, r := syncFixture(t, `{source:"github:o:r:empty",destination:"empty"}`)
	if err := os.WriteFile(filepath.Join(upstream, "empty"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := r.Run(context.Background(), []string{"sync", "diff", "--file", filepath.Join(project, "sync.gh-arrow.cue"), "--config", config}, &out, &stderr); code != 0 || out.String() != "Empty file b/empty is missing locally\n" {
		t.Fatal(code, out.String(), stderr.String())
	}
}
