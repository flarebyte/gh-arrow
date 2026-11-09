package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/listing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const commitSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const rootSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
const subSHA = "cccccccccccccccccccccccccccccccccccccccc"
const blobSHA = "dddddddddddddddddddddddddddddddddddddddd"

func fixture(t *testing.T, routes map[string]string) *Client {
	t.Helper()
	return &Client{HTTP: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		key := r.URL.RequestURI()
		body, ok := routes[key]
		if !ok {
			t.Errorf("unexpected request %s", key)
			return &http.Response{StatusCode: 500, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}"))}, nil
		}
		status := 200
		if body == "404" {
			status = 404
			body = `{"message":"Not Found"}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
}
func repositoryRoutes() map[string]string {
	return map[string]string{
		"/repos/o/r":                             `{"default_branch":"main"}`,
		"/repos/o/r/commits?per_page=1&sha=main": fmt.Sprintf(`[{"sha":%q,"commit":{"tree":{"sha":%q}}}]`, commitSHA, rootSHA),
		"/repos/o/r/git/trees/" + rootSHA:        fmt.Sprintf(`{"tree":[{"path":"a","type":"blob","mode":"100644","sha":%q},{"path":"docs","type":"tree","mode":"040000","sha":%q},{"path":"link","type":"blob","mode":"120000","sha":%q},{"path":"module","type":"commit","mode":"160000","sha":%q}]}`, blobSHA, subSHA, blobSHA, blobSHA),
		"/repos/o/r/git/trees/" + subSHA:         fmt.Sprintf(`{"tree":[{"path":"b.md","type":"blob","mode":"100755","sha":%q}]}`, blobSHA),
		"/repos/o/r/git/blobs/" + blobSHA:        "\x00\xff\r\n",
	}
}

func TestRepositoryPinnedReadsA11A16(t *testing.T) {
	client := fixture(t, repositoryRoutes())
	repo, err := client.Repository(context.Background(), "o/r", "")
	if err != nil {
		t.Fatal(err)
	}
	f, err := repo.Open(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "\x00\xff\r\n" {
		t.Fatalf("%q", data)
	}
	entries, err := repo.List(context.Background(), ".", true)
	if err != nil || len(entries) != 2 {
		t.Fatal(entries, err)
	}
	for _, p := range []string{"link", "module", "docs", "link/target"} {
		if f, err := repo.Open(context.Background(), p); err == nil {
			f.Close()
			t.Errorf("opened %s", p)
		}
	}
	e, err := repo.Stat(context.Background(), "docs/b.md")
	if err != nil || e.Kind != listing.File {
		t.Fatal(e, err)
	}
}

func TestTruncatedTreeA08(t *testing.T) {
	routes := repositoryRoutes()
	routes["/repos/o/r/git/trees/"+rootSHA] = `{"truncated":true,"tree":[]}`
	repo, e := fixture(t, routes).Repository(context.Background(), "o/r", "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = repo.List(context.Background(), ".", true); diagnostic.Code(e) != "ARROW_INCOMPLETE" {
		t.Fatal(e)
	}
}

func TestHTTPFailuresDoNotLeakCredentialsA06(t *testing.T) {
	for _, tt := range []struct {
		status int
		code   string
	}{{401, "ARROW_AUTH"}, {403, "ARROW_PERMISSION"}, {429, "ARROW_RATE_LIMIT"}, {404, "ARROW_NOT_FOUND"}, {500, "ARROW_NETWORK"}, {302, "ARROW_NETWORK"}} {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			calls := 0
			c := &Client{token: "secret-token", HTTP: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != "api.github.com" || r.Header.Get("Authorization") != "Bearer secret-token" {
					t.Fatal("wrong auth destination")
				}
				return &http.Response{StatusCode: tt.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("secret-token from server"))}, nil
			})}}
			_, err := c.Repository(context.Background(), "o/r", "")
			if diagnostic.Code(err) != tt.code || strings.Contains(err.Error(), "secret-token") {
				t.Fatal(err)
			}
			if calls > 3 {
				t.Fatal("unbounded retries")
			}
		})
	}
}

func TestDiscoveryPrivateUserRepositoriesA09A10A17(t *testing.T) {
	routes := repositoryRoutes()
	routes["/users/o"] = `{"type":"User"}`
	routes["/users/o/repos?per_page=100&page=1&type=owner"] = `[]`
	routes["/user/repos?per_page=100&page=1&affiliation=owner,collaborator,organization_member&visibility=all"] = `[{"full_name":"o/r","topics":["cli"],"private":true},{"full_name":"other/r","topics":["cli"]}]`
	c := fixture(t, routes)
	c.token = "test"
	got, e := c.Discover(context.Background(), "o", "cli", "a")
	if e != nil || len(got) != 1 || got[0].Repository != "o/r" {
		t.Fatal(got, e)
	}
	got, e = c.Discover(context.Background(), "o", "cli", "docs")
	if e != nil || len(got) != 0 {
		t.Fatal(got, e)
	}
	got, e = c.Discover(context.Background(), "o", "absent", "a")
	if e != nil || len(got) != 0 {
		t.Fatal(got, e)
	}
}

func TestRefEncodedAsData(t *testing.T) {
	routes := repositoryRoutes()
	routes["/repos/o/r/commits?per_page=1&sha=feature%2Fa%23b%26c"] = routes["/repos/o/r/commits?per_page=1&sha=main"]
	if _, e := fixture(t, routes).Repository(context.Background(), "o/r", "feature/a#b&c"); e != nil {
		t.Fatal(e)
	}
}

func TestEmptyRepository(t *testing.T) {
	c := &Client{HTTP: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		body := `{"default_branch":"main"}`
		status := 200
		if strings.HasSuffix(r.URL.Path, "/commits") {
			status = 409
			body = `{"message":"Git Repository is empty."}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	repo, err := c.Repository(context.Background(), "o/r", "")
	if err != nil {
		t.Fatal(err)
	}
	files, err := repo.List(context.Background(), ".", true)
	if err != nil || len(files) != 0 {
		t.Fatal(files, err)
	}
	if _, err = c.Repository(context.Background(), "o/r", "missing-ref"); err == nil {
		t.Fatal("explicit ref accepted for empty repository")
	}
}

func TestPaginationAndFailureA06A09(t *testing.T) {
	routes := repositoryRoutes()
	routes["/users/o"] = `{"type":"Organization"}`
	repos := make([]string, 100)
	for i := range repos {
		repos[i] = `{"full_name":"o/r","topics":["cli"]}`
	}
	routes["/orgs/o/repos?per_page=100&page=1&type=all"] = "[" + strings.Join(repos, ",") + "]"
	routes["/orgs/o/repos?per_page=100&page=2&type=all"] = `[]`
	client := fixture(t, routes)
	matches, err := client.Discover(context.Background(), "o", "cli", "a")
	if err != nil || len(matches) != 1 {
		t.Fatal(matches, err)
	}
	routes["/orgs/o/repos?per_page=100&page=2&type=all"] = "404"
	if matches, err = client.Discover(context.Background(), "o", "cli", "a"); err == nil || len(matches) != 0 {
		t.Fatal("partial discovery result", matches, err)
	}
}

func TestCanceledRequestsAndMalformedJSON(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := &Client{HTTP: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })}}
	if _, err := c.Repository(ctx, "o/r", ""); err == nil {
		t.Fatal("cancellation ignored")
	}
	routes := repositoryRoutes()
	routes["/repos/o/r"] = "not-json"
	if _, err := fixture(t, routes).Repository(context.Background(), "o/r", ""); diagnostic.Code(err) != "ARROW_NETWORK" {
		t.Fatal(err)
	}
}

func TestMissingTreePayloadIsNotEmptySuccess(t *testing.T) {
	routes := repositoryRoutes()
	routes["/repos/o/r/git/trees/"+rootSHA] = `{}`
	repo, err := fixture(t, routes).Repository(context.Background(), "o/r", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.List(context.Background(), ".", false); diagnostic.Code(err) != "ARROW_NETWORK" {
		t.Fatal(err)
	}
}

func TestTreeCacheBoundsCumulativeBytes(t *testing.T) {
	// Multiple individually legal responses must not grow an unbounded cache.
	// Large virtual names simulate hostile repository metadata without file reads.
	body := fmt.Sprintf(`{"tree":[{"path":%q,"type":"blob","mode":"100644","sha":%q}]}`, strings.Repeat("x", 4<<20), blobSHA)
	client := &Client{HTTP: &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	repo := &Repository{client: client, name: "o/r", trees: make(map[string][]treeEntry)}
	for i := range 17 {
		_, err := repo.tree(context.Background(), fmt.Sprint(i))
		if err != nil {
			if diagnostic.Code(err) != "ARROW_INCOMPLETE" {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatal("cache accepted more than 64 MiB of metadata")
}
