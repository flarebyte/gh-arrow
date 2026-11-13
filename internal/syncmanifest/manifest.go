// Package syncmanifest validates the project-owned one-way file sync manifest.
package syncmanifest

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"
	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/source"
)

type File struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}
type Manifest struct {
	Version int    `json:"version"`
	Files   []File `json:"files"`
}

const schema = `#Manifest: close({version: 1, files: [...close({source: string, destination: string})]})`

func invalid() error {
	return diagnostic.New("ARROW_CONFIG", "invalid sync manifest; expected version 1 and exact file mappings with unique project-relative destinations")
}

func Load(name string) (Manifest, error) {
	var m Manifest
	f, err := os.Open(name)
	if err != nil {
		return m, diagnostic.New("ARROW_CONFIG", "cannot read sync manifest")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 1024*1024+1))
	if err != nil || len(data) > 1024*1024 {
		return m, invalid()
	}
	file, err := parser.ParseFile("sync", data)
	if err != nil {
		return m, invalid()
	}
	for _, decl := range file.Decls {
		if _, ok := decl.(*ast.ImportDecl); ok {
			return m, invalid()
		}
	}
	ctx := cuecontext.New()
	value := ctx.CompileString(schema).LookupPath(cue.MakePath(cue.Def("Manifest"))).Unify(ctx.BuildFile(file))
	if value.Validate(cue.Concrete(true)) != nil || value.Decode(&m) != nil {
		return m, invalid()
	}
	seen := map[string]bool{}
	for i := range m.Files {
		entry := &m.Files[i]
		s, err := source.Parse(entry.Source)
		if err != nil || path.Clean(s.Path) == "." || strings.HasSuffix(s.Path, "/") || strings.ContainsAny(s.Path, "*?[]") {
			return m, invalid()
		}
		d := entry.Destination
		if source.ValidatePath(d) != nil || d == "" || strings.HasSuffix(d, "/") || !filepath.IsLocal(filepath.FromSlash(d)) {
			return m, invalid()
		}
		d = path.Clean(d)
		if d == "." || strings.EqualFold(d, filepath.Base(name)) {
			return m, invalid()
		}
		// Case-fold to reject collisions even when manifests run on case-insensitive systems.
		key := strings.ToLower(d)
		if seen[key] {
			return m, invalid()
		}
		seen[key] = true
		entry.Destination = d
	}
	for key := range seen {
		for parent := path.Dir(key); parent != "."; parent = path.Dir(parent) {
			if seen[parent] {
				return m, invalid()
			}
		}
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Destination < m.Files[j].Destination })
	return m, nil
}
