package local

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type contextReader struct {
	context.Context
	io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(p)
}

// Stream checks cancellation between reads without buffering the entire file.
func Stream(ctx context.Context, w io.Writer, r io.Reader) error {
	_, err := io.Copy(w, contextReader{ctx, r})
	return err
}

// Copy stages private bytes beside the target and publishes them atomically.
// A hard-link/no-replace rename makes the default refusal race-safe.
func Copy(ctx context.Context, r io.Reader, sourceInfo fs.FileInfo, destination string, force bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := os.OpenRoot(filepath.Dir(destination))
	if err != nil {
		return err
	}
	defer dir.Close()
	return copyInto(ctx, dir, r, sourceInfo, filepath.Base(destination), force)
}

// CopyWithin publishes relative to a pinned project root. Parent symlinks are rejected.
func CopyWithin(ctx context.Context, root *os.Root, r io.Reader, destination string) error {
	if err := CheckParent(root, destination); err != nil {
		return err
	}
	dir, err := root.OpenRoot(filepath.Dir(filepath.FromSlash(destination)))
	if err != nil {
		return err
	}
	defer dir.Close()
	return copyInto(ctx, dir, r, nil, filepath.Base(destination), true)
}

// CheckParent requires existing directory components without following symlinks.
func CheckParent(root *os.Root, destination string) error {
	if !filepath.IsLocal(filepath.FromSlash(destination)) {
		return fmt.Errorf("destination must be project-relative")
	}
	for parent := filepath.Dir(filepath.FromSlash(destination)); parent != "."; parent = filepath.Dir(parent) {
		info, err := root.Lstat(parent)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("destination parent must be a directory without symlinks")
		}
	}
	return nil
}

func copyInto(ctx context.Context, dir *os.Root, r io.Reader, sourceInfo fs.FileInfo, name string, force bool) error {
	destination := name
	if name == "." || name == string(filepath.Separator) {
		return fmt.Errorf("destination must be a file")
	}
	info, err := dir.Lstat(name)
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("destination must be a regular file, not a symlink or directory")
		}
		if sourceInfo != nil && os.SameFile(sourceInfo, info) {
			return fmt.Errorf("source and destination are the same file")
		}
		if !force {
			return &os.PathError{Op: "copy", Path: destination, Err: fs.ErrExist}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	temp := ".gh-arrow-" + rand.Text()
	file, err := temporaryFile(dir, temp)
	if err != nil {
		return err
	}
	defer dir.Remove(temp)
	err = Stream(ctx, file, r)
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Go 1.25's rooted operations keep publication relative to the pinned
	// parent on every supported OS, without custom platform rename code.
	if force {
		err = dir.Rename(temp, name)
	} else {
		err = dir.Link(temp, name)
	}
	if err != nil {
		return &os.PathError{Op: "publish", Path: destination, Err: err}
	}
	return nil
}
