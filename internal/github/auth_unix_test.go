//go:build linux || darwin

package github

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGHCredentialsAndRedirectBoundary(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
test "$*" = 'auth token --hostname github.com' || exit 1
test "$GH_PROMPT_DISABLED" = 1 || exit 1
printf 'fixture-secret\n'
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_HOST", "enterprise.example")
	c, err := New(context.Background())
	if err != nil || c.token != "fixture-secret" {
		t.Fatal("credential lookup failed", err)
	}
	calls := 0
	c.HTTP.Transport = transport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "api.github.com" || r.Header.Get("Authorization") != "Bearer fixture-secret" {
			t.Fatal("credential destination changed")
		}
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://unrelated.example/"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	_, err = c.Repository(context.Background(), "o/r", "")
	if err == nil || calls != 1 || strings.Contains(err.Error(), "fixture-secret") {
		t.Fatal("redirect boundary failed", calls, err)
	}
}
