// Package config loads read-only working-tree mappings with strict YAML checks.
package config

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/source"
	"go.yaml.in/yaml/v3"
)

type Mapping struct {
	Repo string `yaml:"repo"`
	Root string `yaml:"root"`
}
type group struct {
	Mappings []Mapping `yaml:"mappings"`
}
type Config struct {
	GitHub group `yaml:"github"`
	Local  group `yaml:"local"`
}

func Load(name string, explicit bool) (Config, error) {
	var c Config
	f, err := os.Open(name)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return c, nil
	}
	if err != nil {
		return c, diagnostic.Wrap("ARROW_CONFIG", "read configuration", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 1024*1024+1))
	if err != nil {
		return c, diagnostic.Wrap("ARROW_CONFIG", "read configuration", err)
	}
	if len(data) > 1024*1024 {
		return c, diagnostic.New("ARROW_CONFIG", "configuration exceeds 1 MiB")
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err = dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return c, diagnostic.Wrap("ARROW_CONFIG", "decode configuration", err)
	}
	var extra any
	if err = dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return c, diagnostic.New("ARROW_CONFIG", "configuration must contain a single YAML document")
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
