//go:build linux || darwin

package local

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"bytes"
	"github.com/flarebyte/gh-arrow/internal/diagnostic"

	"golang.org/x/sys/unix"
)

func TestSpecialFileA15(t *testing.T) {
	root := t.TempDir()
	unix.Mkfifo(filepath.Join(root, "fifo"), 0600)
	b, _ := New(root)
	defer b.Close()
	done := make(chan error, 1)
	go func() {
		f, e := b.Open(context.Background(), "fifo")
		if f != nil {
			f.Close()
		}
		done <- e
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO open blocked")
	}
	entries, e := b.List(context.Background(), ".", false)
	if e != nil || len(entries) != 0 {
		t.Fatal(entries, e)
	}
}

func TestConcurrentSymlinkReplacementA12(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "secret"), []byte("outside"), 0600)
	b, _ := New(root)
	defer b.Close()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			os.Symlink(outside, filepath.Join(root, "swap"))
			os.Remove(filepath.Join(root, "swap"))
			os.Mkdir(filepath.Join(root, "swap"), 0700)
			os.Remove(filepath.Join(root, "swap"))
		}
	}()
	defer func() { close(stop); wg.Wait() }()
	for range 300 {
		f, e := b.Open(context.Background(), "swap/secret")
		if e == nil {
			f.Close()
			t.Fatal("opened outside file during replacement")
		}
	}
}

func TestListingRejectsInvalidUTF8Names(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bad\xff"), []byte("bytes"), 0600); err != nil {
		t.Skipf("filesystem rejects invalid UTF-8 fixture: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || utf8.ValidString(entries[0].Name()) {
		t.Skip("filesystem normalizes invalid UTF-8 filenames")
	}
	b, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if _, err = b.List(context.Background(), ".", false); err == nil {
		t.Fatal("listing would silently replace invalid UTF-8 in JSON")
	}
}

func TestUnwritableParentA05(t *testing.T) {
	if unix.Geteuid() == 0 {
		t.Skip("root can bypass directory permissions")
	}
	root := t.TempDir()
	name := filepath.Join(root, "original")
	os.WriteFile(name, []byte("original"), 0600)
	os.Chmod(root, 0000)
	defer os.Chmod(root, 0700)
	err := Copy(context.Background(), bytes.NewBufferString("replacement"), nil, name, true)
	if diagnostic.Code(err) != "ARROW_PERMISSION" {
		t.Fatal(err)
	}
	os.Chmod(root, 0700)
	data, _ := os.ReadFile(name)
	if string(data) != "original" {
		t.Fatal(string(data))
	}
}
