package cli

import (
	"bytes"
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
