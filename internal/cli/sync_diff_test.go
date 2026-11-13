package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flarebyte/gh-arrow/internal/local"
	"testing"
)

func TestUnifiedDiffRangesAndNewlines(t *testing.T) {
	for _, tt := range []struct{ before, after, want string }{
		{"", "new\n", "--- a/x\n+++ b/x\n@@ -0,0 +1,1 @@\n+new\n"},
		{"old\n", "", "--- a/x\n+++ b/x\n@@ -1,1 +0,0 @@\n-old\n"},
		{"old", "new", "--- a/x\n+++ b/x\n@@ -1,1 +1,1 @@\n-old\n\\ No newline at end of file\n+new\n\\ No newline at end of file\n"},
		{"0\n1\n2\n3\nold\n5\n6\n7\n8\n", "0\n1\n2\n3\nnew\n5\n6\n7\n8\n", "--- a/x\n+++ b/x\n@@ -2,7 +2,7 @@\n 1\n 2\n 3\n-old\n+new\n 5\n 6\n 7\n"},
	} {
		var out bytes.Buffer
		if err := unifiedDiff(&out, "a/x", "b/x", []byte(tt.before), []byte(tt.after)); err != nil || out.String() != tt.want {
			t.Fatalf("got %q want %q error %v", out.String(), tt.want, err)
		}
	}
}

// A27: both local and upstream content must be safe to display, including
// controls in otherwise unchanged context around a changed line.
func TestSyncDiffRejectsTerminalControlsA27(t *testing.T) {
	for _, control := range []string{"\x1b[2J", "\a", "\b", "\x7f", "\u009b2J", "\u202e", "\r"} {
		for _, side := range []string{"local", "upstream", "context"} {
			t.Run(side+"/"+fmt.Sprintf("%q", control), func(t *testing.T) {
				project := t.TempDir()
				before, after := "old\n", "new\n"
				switch side {
				case "local":
					before = control + before
				case "upstream":
					after = control + after
				case "context":
					before = control + "context\n" + before
					after = control + "context\n" + after
				}
				if err := os.WriteFile(filepath.Join(project, "file"), []byte(before), 0600); err != nil {
					t.Fatal(err)
				}
				staged := filepath.Join(t.TempDir(), "staged")
				if err := os.WriteFile(staged, []byte(after), 0600); err != nil {
					t.Fatal(err)
				}
				current, err := local.New(project)
				if err != nil {
					t.Fatal(err)
				}
				defer current.Close()
				var out bytes.Buffer
				err = syncDiff(context.Background(), current, []syncItem{{record: syncRecord{Destination: "file", Status: "modified"}, staged: staged}}, &out)
				if err != nil || out.String() != "Binary files a/file and b/file differ\n" {
					t.Fatalf("output=%q error=%v", out.String(), err)
				}
			})
		}
	}
}

func TestSyncDiffPreservesOrdinaryText(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "file"), []byte("old\tvalue\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(t.TempDir(), "staged")
	if err := os.WriteFile(staged, []byte("new\t世界\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	current, err := local.New(project)
	if err != nil {
		t.Fatal(err)
	}
	defer current.Close()
	var out bytes.Buffer
	err = syncDiff(context.Background(), current, []syncItem{{record: syncRecord{Destination: "file", Status: "modified"}, staged: staged}}, &out)
	if err != nil || !bytes.Contains(out.Bytes(), []byte("-old\tvalue\r\n+new\t世界\r\n")) {
		t.Fatalf("output=%q error=%v", out.String(), err)
	}
}
