package cli

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/flarebyte/gh-arrow/internal/config"
	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/github"
	"github.com/flarebyte/gh-arrow/internal/listing"
	"github.com/flarebyte/gh-arrow/internal/local"
	"github.com/flarebyte/gh-arrow/internal/source"
	"github.com/flarebyte/gh-arrow/internal/syncmanifest"
)

type syncRecord struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Status      string `json:"status"`
	Backend     string `json:"backend"`
}
type syncItem struct {
	record syncRecord
	staged string
}

func (r Runner) sync(ctx context.Context, o options, c config.Config, out io.Writer) error {
	m, err := syncmanifest.Load(o.file)
	if err != nil {
		return err
	}
	project := filepath.Dir(o.file)
	root, err := os.OpenRoot(project)
	if err != nil {
		return err
	}
	defer root.Close()
	current, err := local.New(project)
	if err != nil {
		return err
	}
	defer current.Close()
	staging, err := os.MkdirTemp("", "gh-arrow-sync-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	// Cache backends so each remote repository resolves its default branch once.
	backends := map[string]backend{}
	var client *github.Client
	var locals []*local.Backend
	defer func() {
		for _, b := range locals {
			b.Close()
		}
	}()
	items := make([]syncItem, 0, len(m.Files))
	for i, entry := range m.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := local.CheckParent(root, entry.Destination); err != nil {
			return fmt.Errorf("plan %s: %w", entry.Destination, err)
		}
		s, _ := source.Parse(entry.Source)
		mapped, err := c.Root(s)
		if err != nil {
			return err
		}
		kind, key := "github", "github:"+strings.ToLower(s.Repository)
		if mapped != "" {
			kind, key = "local", "local:"+mapped
		}
		b := backends[key]
		if b == nil {
			if mapped != "" {
				lb, err := local.New(mapped)
				if err != nil {
					return err
				}
				locals = append(locals, lb)
				b = lb
			} else {
				if client == nil {
					client, err = r.remote(ctx)
					if err != nil {
						return err
					}
				}
				b, err = client.Repository(ctx, s.Repository, "")
				if err != nil {
					return err
				}
			}
			backends[key] = b
		}
		file, err := b.Open(ctx, s.Path)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", entry.Destination, err)
		}
		staged := filepath.Join(staging, fmt.Sprint(i))
		digest, err := stageSync(ctx, staged, file)
		if err != nil {
			return fmt.Errorf("stage %s: %w", entry.Destination, err)
		}
		status := "missing"
		existing, err := current.Open(ctx, entry.Destination)
		if err == nil {
			h := sha256.New()
			readErr := local.Stream(ctx, h, existing)
			closeErr := existing.Close()
			if readErr != nil {
				return readErr
			}
			if closeErr != nil {
				return closeErr
			}
			status = "modified"
			if string(h.Sum(nil)) == digest {
				status = "unchanged"
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("read %s: %w", entry.Destination, err)
		} else {
			// A dangling symlink is not a missing regular file.
			if _, statErr := root.Lstat(filepath.FromSlash(entry.Destination)); statErr == nil {
				return fmt.Errorf("destination must be a regular file")
			}
		}
		items = append(items, syncItem{syncRecord{entry.Source, entry.Destination, status, kind}, staged})
	}
	if o.syncAction == "diff" {
		return syncDiff(ctx, current, items, out)
	}
	if o.syncAction == "" && !o.dryRun {
		applied := 0
		for _, item := range items {
			if item.record.Status == "unchanged" {
				continue
			}
			f, err := os.Open(item.staged)
			if err == nil {
				err = local.CopyWithin(ctx, root, f, item.record.Destination)
				f.Close()
			}
			if err != nil {
				return fmt.Errorf("apply %s after %d file(s) applied; partial application possible: %w", item.record.Destination, applied, err)
			}
			applied++
		}
	}
	records := make([]syncRecord, 0, len(items))
	drift := false
	for _, item := range items {
		records = append(records, item.record)
		drift = drift || item.record.Status != "unchanged"
	}
	if o.json {
		err = writeJSON(out, records)
	} else {
		for _, record := range records {
			if _, err = fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", record.Status, listing.Escape(record.Destination), record.Backend, listing.Escape(record.Source)); err != nil {
				break
			}
		}
	}
	if err != nil {
		return err
	}
	if o.syncAction == "check" && drift {
		return diagnostic.New("ARROW_DRIFT", "declared files differ from upstream")
	}
	return nil
}

func stageSync(ctx context.Context, name string, source io.ReadCloser) (string, error) {
	defer source.Close()
	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	err = local.Stream(ctx, io.MultiWriter(f, h), source)
	closeErr := f.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	return string(h.Sum(nil)), nil
}
