// Package cli orchestrates operations; backends own filesystem and HTTP details.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"runtime/debug"

	project "github.com/flarebyte/gh-arrow"

	"github.com/flarebyte/gh-arrow/internal/config"
	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/github"
	"github.com/flarebyte/gh-arrow/internal/listing"
	"github.com/flarebyte/gh-arrow/internal/local"
	"github.com/flarebyte/gh-arrow/internal/source"
)

// Only this consumer needs a shared backend interface.
type backend interface {
	Open(context.Context, string) (io.ReadCloser, error)
	Stat(context.Context, string) (listing.Entry, error)
	List(context.Context, string, bool) ([]listing.Entry, error)
}
type Runner struct {
	NewRemote func(context.Context) (*github.Client, error)
}

func (r Runner) remote(ctx context.Context) (*github.Client, error) {
	if r.NewRemote != nil {
		return r.NewRemote(ctx)
	}
	return github.New(ctx)
}
func (r Runner) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	o, err := parse(args)
	if err == nil {
		err = r.execute(ctx, o, stdout)
	}
	if err != nil {
		if o.command != "" {
			err = fmt.Errorf("%s: %w", o.command, err)
		}
		fmt.Fprintf(stderr, "%s: %s\n", diagnostic.Code(err), listing.Escape(err.Error()))
	}
	return diagnostic.Exit(err)
}
func (r Runner) execute(ctx context.Context, o options, out io.Writer) error {
	if o.version {
		text := "gh-arrow " + project.Version()
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					text += " (commit " + setting.Value + ")"
				}
				if setting.Key == "vcs.modified" && setting.Value == "true" {
					text += " (modified)"
				}
			}
		}
		_, err := fmt.Fprintln(out, text)
		return err
	}
	if o.help {
		_, err := fmt.Fprint(out, "Usage: gh arrow <command> [options]\n\nCommands:\n  cp <source> <destination> [--force]\n  cat <source>\n  ls <source> [--json]\n  ls --org <owner> [--topic <topic>] <path> [--json]\n\nRoot option: --version\nOptions: --config <path>, --ref <ref> (remote sources only), --help\nSources: github:<owner>:<repo>:<path> or local:<alias>:<path>\n\nExamples:\n  gh arrow cat local:project:README.md\n  gh arrow ls 'github:owner:repo:docs/**/*.md' --json\n  gh arrow cp github:owner:repo:README.md ./README.md\n")
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	c, err := config.Load(o.config, o.explicit)
	if err != nil {
		return err
	}
	if o.owner != "" {
		client, err := r.remote(ctx)
		if err != nil {
			return err
		}
		matches, err := client.Discover(ctx, o.owner, o.topic, o.args[0])
		if err != nil {
			return err
		}
		if o.json {
			return writeJSON(out, matches)
		}
		for _, m := range matches {
			if _, err := fmt.Fprintf(out, "%s:%s\n", listing.Escape(m.Repository), listing.Escape(m.Path)); err != nil {
				return err
			}
		}
		return nil
	}
	s, _ := source.Parse(o.args[0])
	root, err := c.Root(s)
	if err != nil {
		return err
	}
	var b backend
	if root != "" {
		if o.ref != "" {
			return usage("--ref is not supported for mapped sources")
		}
		localBackend, err := local.New(root)
		if err != nil {
			return err
		}
		defer localBackend.Close()
		b = localBackend
	} else {
		client, err := r.remote(ctx)
		if err != nil {
			return err
		}
		b, err = client.Repository(ctx, s.Repository, o.ref)
		if err != nil {
			return err
		}
	}
	if o.command == "ls" {
		return list(ctx, b, s.Path, o.json, out)
	}
	file, err := b.Open(ctx, s.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	if o.command == "cat" {
		return local.Stream(ctx, out, file)
	}
	var info fs.FileInfo
	if stat, ok := file.(interface{ Stat() (fs.FileInfo, error) }); ok {
		info, err = stat.Stat()
		if err != nil {
			return err
		}
	}
	return local.Copy(ctx, file, info, o.args[1], o.force)
}
func list(ctx context.Context, b backend, name string, asJSON bool, out io.Writer) error {
	p, err := listing.Parse(name)
	if err != nil {
		return err
	}
	paths := []string{}
	if !p.Glob {
		entry, err := b.Stat(ctx, p.Path)
		if err != nil {
			return err
		}
		if entry.Kind == listing.File {
			paths = append(paths, path.Base(p.Path))
		} else if entry.Kind == listing.Directory {
			entries, err := b.List(ctx, p.Path, false)
			if err != nil {
				return err
			}
			paths = p.Select(entries)
		}
	} else {
		entry, statErr := b.Stat(ctx, p.Base)
		if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
			return statErr
		}
		if statErr == nil && entry.Kind == listing.Directory {
			entries, err := b.List(ctx, p.Base, p.Recursive)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			paths = p.Select(entries)
		}
	}
	if asJSON {
		records := make([]struct {
			Path string `json:"path"`
		}, len(paths))
		for i, p := range paths {
			records[i].Path = p
		}
		return writeJSON(out, records)
	}
	for _, p := range paths {
		if _, err := fmt.Fprintln(out, listing.Escape(p)); err != nil {
			return err
		}
	}
	return nil
}
func writeJSON(out io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = out.Write(append(data, '\n'))
	return err
}
