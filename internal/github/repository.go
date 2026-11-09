package github

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strings"

	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/listing"
	"github.com/flarebyte/gh-arrow/internal/source"
)

type treeEntry struct{ Path, Mode, Type, SHA string }
type tree struct {
	Tree      []treeEntry
	Truncated bool
}
type Repository struct {
	client          *Client
	name, root      string
	trees           map[string][]treeEntry
	requests        int
	metadataEntries int
}

func (c *Client) Repository(ctx context.Context, name, ref string) (*Repository, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || !source.ValidRepository(parts[0], parts[1]) {
		return nil, diagnostic.New("ARROW_USAGE", "invalid repository identity")
	}
	var meta struct {
		DefaultBranch *string `json:"default_branch"`
	}
	if err := c.json(ctx, "/repos/"+name, &meta); err != nil {
		return nil, err
	}
	if meta.DefaultBranch == nil {
		return nil, diagnostic.New("ARROW_NETWORK", "GitHub response is missing its default branch")
	}
	repo := &Repository{client: c, name: name, trees: make(map[string][]treeEntry)}
	defaultRef := ref == ""
	if defaultRef {
		ref = *meta.DefaultBranch
	}
	if ref == "" {
		return repo, nil
	}
	var commits []struct {
		SHA    string
		Commit struct{ Tree struct{ SHA string } }
	}
	// Unlike the single-commit endpoint, this returns no file diffs or patches.
	if err := c.json(ctx, "/repos/"+name+"/commits?per_page=1&sha="+url.QueryEscape(ref), &commits); err != nil {
		var status httpStatus
		if defaultRef && errors.As(err, &status) && status == 409 {
			return repo, nil
		}
		return nil, err
	}
	if len(commits) != 1 || commits[0].SHA == "" || commits[0].Commit.Tree.SHA == "" {
		return nil, diagnostic.New("ARROW_NETWORK", "GitHub did not return a commit tree")
	}
	repo.root = commits[0].Commit.Tree.SHA
	return repo, nil
}
func kind(e treeEntry) listing.Kind {
	switch {
	case e.Mode == "120000":
		return listing.Symlink
	case e.Type == "commit" || e.Mode == "160000":
		return listing.Submodule
	case e.Type == "tree":
		return listing.Directory
	case e.Type == "blob" && (e.Mode == "100644" || e.Mode == "100755"):
		return listing.File
	default:
		return listing.Submodule
	}
}
func (r *Repository) tree(ctx context.Context, sha string) ([]treeEntry, error) {
	if sha == "" {
		return []treeEntry{}, nil
	}
	if t, ok := r.trees[sha]; ok {
		return t, nil
	}
	r.requests++
	if r.requests > 10000 {
		return nil, diagnostic.New("ARROW_INCOMPLETE", "repository exceeds 10000 tree requests")
	}
	var t tree
	if err := r.client.json(ctx, "/repos/"+r.name+"/git/trees/"+url.PathEscape(sha), &t); err != nil {
		return nil, err
	}
	if t.Truncated {
		return nil, diagnostic.New("ARROW_INCOMPLETE", "GitHub tree response is truncated")
	}
	if t.Tree == nil {
		return nil, diagnostic.New("ARROW_NETWORK", "GitHub response is missing its tree entries")
	}
	r.metadataEntries += len(t.Tree)
	if r.metadataEntries > 1000000 {
		return nil, diagnostic.New("ARROW_INCOMPLETE", "repository exceeds one million metadata entries")
	}
	for _, entry := range t.Tree {
		if entry.Path == "" || strings.ContainsAny(entry.Path, "/\\") || entry.Path == "." || entry.Path == ".." || entry.SHA == "" {
			return nil, diagnostic.New("ARROW_NETWORK", "invalid GitHub tree entry")
		}
	}
	r.trees[sha] = t.Tree
	return t.Tree, nil
}
func (r *Repository) lookup(ctx context.Context, name string) (treeEntry, error) {
	if err := source.ValidatePath(name); err != nil {
		return treeEntry{}, err
	}
	name = path.Clean(name)
	current := treeEntry{Type: "tree", Mode: "040000", SHA: r.root}
	if name == "." {
		return current, nil
	}
	for _, part := range strings.Split(name, "/") {
		if kind(current) != listing.Directory {
			return treeEntry{}, fs.ErrNotExist
		}
		entries, err := r.tree(ctx, current.SHA)
		if err != nil {
			return treeEntry{}, err
		}
		found := false
		for _, e := range entries {
			if e.Path == part {
				current = e
				found = true
				break
			}
		}
		if !found {
			return treeEntry{}, fs.ErrNotExist
		}
	}
	return current, nil
}
func (r *Repository) Stat(ctx context.Context, name string) (listing.Entry, error) {
	e, err := r.lookup(ctx, name)
	return listing.Entry{Path: path.Clean(name), Kind: kind(e)}, err
}
func (r *Repository) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	e, err := r.lookup(ctx, name)
	if err != nil {
		return nil, err
	}
	if kind(e) != listing.File {
		return nil, diagnostic.New("ARROW_IO", "source must be a regular file, not a symlink, submodule or directory")
	}
	response, err := r.client.get(ctx, "/repos/"+r.name+"/git/blobs/"+url.PathEscape(e.SHA), "application/vnd.github.raw+json")
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}
func (r *Repository) List(ctx context.Context, name string, recursive bool) ([]listing.Entry, error) {
	start, err := r.lookup(ctx, name)
	if err != nil {
		return nil, err
	}
	if kind(start) != listing.Directory {
		return nil, diagnostic.New("ARROW_IO", "listing prefix must be a directory")
	}
	result := []listing.Entry{}
	visited := 0
	var walk func(string, string, int) error
	walk = func(dir, sha string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if depth > 256 || len(result) > 1000000 {
			return diagnostic.New("ARROW_INCOMPLETE", "repository exceeds listing depth or entry limit")
		}
		entries, err := r.tree(ctx, sha)
		if err != nil {
			return err
		}
		for _, e := range entries {
			visited++
			if visited > 1000000 {
				return diagnostic.New("ARROW_INCOMPLETE", "repository exceeds one million traversed entries")
			}
			p := path.Join(dir, e.Path)
			if kind(e) == listing.File {
				if len(result) >= 1000000 {
					return diagnostic.New("ARROW_INCOMPLETE", "repository exceeds one million listing entries")
				}
				result = append(result, listing.Entry{Path: p, Kind: listing.File})
			} else if recursive && kind(e) == listing.Directory {
				if err := walk(p, e.SHA, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(path.Clean(name), start.SHA, 0); err != nil {
		return nil, err
	}
	return result, nil
}

func isMissing(err error) bool { return errors.Is(err, fs.ErrNotExist) }
