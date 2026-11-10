// Package config loads read-only working-tree mappings with an embedded CUE schema.
package config

import (
	_ "embed"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"
	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/source"
)

type Mapping struct {
	Repo string `json:"repo"`
	Root string `json:"root"`
}
type group struct {
	Mappings []Mapping `json:"mappings"`
}
type Config struct {
	GitHub group `json:"github"`
	Local  group `json:"local"`
}

//go:embed schema.cue
var schema string

func Load(name string, explicit bool) (Config, error) {
	var c Config
	if !explicit && name == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return c, diagnostic.New("ARROW_CONFIG", "cannot resolve home directory")
		}
		name = filepath.Join(home, ".gh-arrow")
	}
	f, err := os.Open(name)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return c, nil
	}
	if err != nil {
		return c, diagnostic.New("ARROW_CONFIG", "cannot read configuration")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 1024*1024+1))
	if err != nil {
		return c, diagnostic.New("ARROW_CONFIG", "cannot read configuration")
	}
	if len(data) > 1024*1024 {
		return c, diagnostic.New("ARROW_CONFIG", "configuration exceeds 1 MiB")
	}
	file, err := parser.ParseFile("config", data)
	if err != nil {
		return c, diagnostic.New("ARROW_CONFIG", "invalid CUE configuration; imports are not allowed")
	}
	for _, decl := range file.Decls {
		if _, ok := decl.(*ast.ImportDecl); ok {
			return c, diagnostic.New("ARROW_CONFIG", "CUE imports are not allowed")
		}
	}
	ctx := cuecontext.New()
	value := ctx.CompileString(schema).LookupPath(cue.MakePath(cue.Def("Config"))).Unify(ctx.BuildFile(file))
	if err := value.Validate(cue.Concrete(true)); err != nil {
		return c, diagnostic.New("ARROW_CONFIG", "invalid CUE configuration; check schema and concrete values")
	}
	if err := value.Decode(&c); err != nil {
		return c, diagnostic.New("ARROW_CONFIG", "invalid CUE configuration")
	}
	for i, g := range []group{c.GitHub, c.Local} {
		seen := map[string]bool{}
		for _, m := range g.Mappings {
			key := m.Repo
			valid := source.ValidAlias(key)
			if i == 0 {
				p := strings.Split(key, ":")
				valid = len(p) == 2 && source.ValidRepository(p[0], p[1])
				key = strings.ToLower(key)
			}
			if !valid || !filepath.IsAbs(m.Root) || seen[key] {
				return c, diagnostic.New("ARROW_CONFIG", "invalid or duplicate mapping; roots must be absolute")
			}
			seen[key] = true
		}
	}
	return c, nil
}
func (c Config) Root(s source.Source) (string, error) {
	if s.Scheme == "github" {
		key := strings.ReplaceAll(s.Repository, "/", ":")
		for _, m := range c.GitHub.Mappings {
			if strings.EqualFold(m.Repo, key) {
				return m.Root, nil
			}
		}
		return "", nil
	}
	for _, m := range c.Local.Mappings {
		if m.Repo == s.Alias {
			return m.Root, nil
		}
	}
	return "", diagnostic.New("ARROW_CONFIG", "local alias has no configured mapping")
}
