package local

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/listing"
)

func TestReadAndListA01A07A12(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "docs"), 0700)
	want := []byte{0, 255, '\r', '\n'}
	os.WriteFile(filepath.Join(root, "docs", "a"), want, 0600)
	b, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	ctx := context.Background()
	f, err := b.Open(ctx, "docs/a")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(f)
	f.Close()
	if !bytes.Equal(got, want) {
		t.Fatal(got)
	}
	if _, err = b.Open(ctx, "../outside"); err == nil {
		t.Fatal("traversal accepted")
	}
	if _, err = b.Open(ctx, "docs"); err == nil {
		t.Fatal("directory source accepted")
	}
	if _, err = b.Open(ctx, "missing"); diagnostic.Code(err) != "ARROW_NOT_FOUND" {
		t.Fatal(err)
	}
	entries, err := b.List(ctx, "docs", true)
	if err != nil || !reflect.DeepEqual(entries, []listing.Entry{{Path: "docs/a", Kind: listing.File}}) {
		t.Fatal(entries, err)
	}
	outside := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(outside, []byte("secret"), 0600)
	if err = os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err = b.Open(ctx, "escape"); err == nil {
		t.Fatal("symlink accepted")
	}
	os.Symlink("docs", filepath.Join(root, "alias"))
	if _, err = b.Open(ctx, "alias/a"); err == nil {
		t.Fatal("symlink component accepted")
	}
	entries, err = b.List(ctx, ".", true)
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
}

func TestConcurrentNoClobberA05(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "out")
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, text := range []string{"first", "second"} {
		wg.Add(1)
		go func(text string) {
			defer wg.Done()
			<-start
			results <- Copy(context.Background(), bytes.NewBufferString(text), nil, destination, false)
		}(text)
	}
	close(start)
	wg.Wait()
	close(results)
	success, exists := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if diagnostic.Code(err) == "ARROW_EXISTS" {
			exists++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || exists != 1 {
		t.Fatal(success, exists)
	}
}

type failedReader struct{}

func (failedReader) Read(p []byte) (int, error) {
	copy(p, "partial")
	return 7, errors.New("transfer failed")
}

func TestCopyA05A14(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "out")
	os.WriteFile(dest, []byte("original"), 0600)
	ctx := context.Background()
	if err := Copy(ctx, bytes.NewBufferString("new"), nil, dest, false); diagnostic.Code(err) != "ARROW_EXISTS" {
		t.Fatal(err)
	}
	if err := Copy(ctx, failedReader{}, nil, dest, true); err == nil {
		t.Fatal("transfer failure ignored")
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "original" {
		t.Fatal(string(got))
	}
	if err := Copy(ctx, bytes.NewBufferString("new"), nil, dest, true); err != nil {
		t.Fatal(err)
	}
	f, _ := os.Open(dest)
	info, _ := f.Stat()
	if err := Copy(ctx, f, info, dest, true); err == nil {
		t.Fatal("same file allowed")
	}
	f.Close()
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Fatal("temporary files leaked", entries)
	}
	if err := Copy(ctx, bytes.NewBufferString("x"), nil, filepath.Join(root, "absent", "x"), false); err == nil {
		t.Fatal("created parent")
	}
	if err := Copy(ctx, bytes.NewBufferString("x"), nil, filepath.Join(root, "new"), false); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dest, filepath.Join(root, "link")); err == nil {
		if err := Copy(ctx, bytes.NewBufferString("x"), nil, filepath.Join(root, "link"), true); err == nil {
			t.Fatal("destination symlink allowed")
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if err := Copy(ctx, bytes.NewBufferString("x"), nil, dest, true); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
