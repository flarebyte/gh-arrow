// Package local provides descriptor-relative, non-following working-tree reads.
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/listing"
	"github.com/flarebyte/gh-arrow/internal/source"
)

type Backend struct {
	root      *os.Root
	directory *os.File
}

func New(name string) (*Backend, error) {
	root, err := os.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	dir, err := root.Open(".")
	if err != nil {
		root.Close()
		return nil, err
	}
	return &Backend{root, dir}, nil
}
func (b *Backend) Close() error { defer b.root.Close(); return b.directory.Close() }
func (b *Backend) open(ctx context.Context, name string) (*os.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := source.ValidatePath(name); err != nil {
		return nil, err
	}
	name = path.Clean(name)
	if name == "" {
		name = "."
	}
	if !filepath.IsLocal(filepath.FromSlash(name)) {
		return nil, fmt.Errorf("invalid local path")
	}
	return openRelative(b.directory, name)
}
func (b *Backend) Stat(ctx context.Context, name string) (listing.Entry, error) {
	if err := source.ValidatePath(name); err != nil {
		return listing.Entry{}, err
	}
	name = path.Clean(name)
	parent, err := b.open(ctx, path.Dir(name))
	if err != nil {
		return listing.Entry{}, err
	}
	parent.Close()
	info, err := b.root.Lstat(filepath.FromSlash(name))
	if err != nil {
		return listing.Entry{}, err
	}
	kind := listing.Symlink
	if info.Mode().IsRegular() {
		kind = listing.File
	} else if info.IsDir() {
		kind = listing.Directory
	}
	return listing.Entry{Path: name, Kind: kind}, nil
}
func (b *Backend) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	f, err := b.open(ctx, name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		f.Close()
		return nil, fmt.Errorf("source must be a regular file")
	}
	return f, nil
}
func (b *Backend) List(ctx context.Context, name string, recursive bool) ([]listing.Entry, error) {
	result := []listing.Entry{}
	visited := 0
	var walk func(string) error
	walk = func(dir string) error {
		if strings.Count(dir, "/") > 256 {
			return diagnostic.New("ARROW_INCOMPLETE", "local listing exceeds directory depth limit")
		}
		f, err := b.open(ctx, dir)
		if err != nil {
			return err
		}
		defer f.Close()
		for {
			entries, err := f.ReadDir(256)
			if err != nil && err != io.EOF {
				return err
			}
			for _, e := range entries {
				visited++
				if visited > 1000000 {
					return diagnostic.New("ARROW_INCOMPLETE", "local listing exceeds one million traversed entries")
				}
				if !utf8.ValidString(e.Name()) {
					return diagnostic.New("ARROW_IO", "filename cannot be represented as UTF-8; listing aborted")
				}
				if err := ctx.Err(); err != nil {
					return err
				}
				p := path.Join(dir, e.Name())
				info, err := b.root.Lstat(filepath.FromSlash(p))
				if err != nil {
					return err
				}
				if info.Mode().IsRegular() {
					if len(result) >= 1000000 {
						return diagnostic.New("ARROW_INCOMPLETE", "local listing exceeds one million entries")
					}
					result = append(result, listing.Entry{Path: p, Kind: listing.File})
				} else if info.IsDir() && recursive {
					if err := walk(p); err != nil {
						return err
					}
				}
			}
			if err == io.EOF {
				break
			}
		}
		return nil
	}
	if err := walk(path.Clean(name)); err != nil {
		return nil, err
	}
	return result, nil
}
