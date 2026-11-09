package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flarebyte/gh-arrow/internal/github"
)

type httpTransport func(*http.Request) (*http.Response, error)

func (f httpTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func remoteFixture(t *testing.T, routes map[string]string) Runner {
	t.Helper()
	return Runner{NewRemote: func(context.Context) (*github.Client, error) {
		return &github.Client{HTTP: &http.Client{Transport: httpTransport(func(r *http.Request) (*http.Response, error) {
			body, ok := routes[r.URL.RequestURI()]
			if !ok {
				t.Errorf("unexpected request %s", r.URL.RequestURI())
				body = "RATE"
			}
			status := 200
			if body == "RATE" {
				status = 429
				body = `{"message":"rate limit"}`
			}
			return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		})}}, nil
	}}
}
func TestRemoteCommandContractsA06A11A16(t *testing.T) {
	_, config := localFixture(t)
	routes := map[string]string{
		"/repos/remote/repo":                             `{"default_branch":"main"}`,
		"/repos/remote/repo/commits?per_page=1&sha=main": `[{"sha":"commit","commit":{"tree":{"sha":"root"}}}]`,
		"/repos/remote/repo/git/trees/root":              `{"tree":[{"path":"a","type":"blob","mode":"100644","sha":"file"},{"path":"link","type":"blob","mode":"120000","sha":"file"},{"path":"docs","type":"tree","mode":"040000","sha":"sub"}]}`,
		"/repos/remote/repo/git/trees/sub":               `{"tree":[{"path":"b","type":"blob","mode":"100644","sha":"file"}]}`,
		"/repos/remote/repo/git/blobs/file":              "\x00\xff\r\n",
	}
	runner := remoteFixture(t, routes)
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"cat", "github:remote:repo:a"}, "\x00\xff\r\n"},
		{[]string{"ls", "github:remote:repo:"}, "a\n"},
		{[]string{"ls", "github:remote:repo:**", "--json"}, "[{\"path\":\"a\"},{\"path\":\"docs/b\"}]\n"},
		{[]string{"ls", "github:remote:repo:link", "--json"}, "[]\n"},
	} {
		var out, err bytes.Buffer
		if code := runner.Run(context.Background(), append(tt.args, "--config", config), &out, &err); code != 0 || out.String() != tt.want || err.Len() != 0 {
			t.Fatalf("%v: %d %q %q", tt.args, code, out.String(), err.String())
		}
	}
	dest := filepath.Join(t.TempDir(), "copy")
	var out, err bytes.Buffer
	if code := runner.Run(context.Background(), []string{"cp", "github:remote:repo:a", dest, "--config", config}, &out, &err); code != 0 || out.Len() != 0 {
		t.Fatal(code, err.String())
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "\x00\xff\r\n" {
		t.Fatalf("%q", data)
	}
	routes["/repos/remote/repo/git/trees/sub"] = "RATE"
	out.Reset()
	err.Reset()
	if code := runner.Run(context.Background(), []string{"ls", "github:remote:repo:**", "--config", config}, &out, &err); code != 1 || out.Len() != 0 || !strings.HasPrefix(err.String(), "ARROW_RATE_LIMIT:") {
		t.Fatal(code, out.String(), err.String())
	}
}

func TestDiscoveryIgnoresMappingsA10A13(t *testing.T) {
	_, config := localFixture(t)
	runner := remoteFixture(t, map[string]string{
		"/users/o": `{"type":"Organization"}`,
		"/orgs/o/repos?per_page=100&page=1&type=all": `[{"full_name":"o/r","topics":["cli"]}]`,
		"/repos/o/r":                             `{"default_branch":"main"}`,
		"/repos/o/r/commits?per_page=1&sha=main": `[{"sha":"commit","commit":{"tree":{"sha":"root"}}}]`,
		"/repos/o/r/git/trees/root":              `{"tree":[{"path":"README.md","type":"blob","mode":"100644","sha":"file"}]}`,
	})
	var out, err bytes.Buffer
	code := runner.Run(context.Background(), []string{"ls", "--org", "o", "--topic", "cli", "README.md", "--config", config, "--json"}, &out, &err)
	if code != 0 || out.String() != "[{\"repository\":\"o/r\",\"path\":\"README.md\"}]\n" {
		t.Fatal(code, out.String(), err.String())
	}
}
