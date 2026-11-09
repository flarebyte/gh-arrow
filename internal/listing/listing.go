// Package listing implements the same small glob grammar for both backends.
package listing

import (
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/source"
)

type Kind uint8

const (
	File Kind = iota
	Directory
	Symlink
	Submodule
)

type Entry struct {
	Path string
	Kind Kind
}
type Pattern struct {
	Path, Base      string
	Glob, Recursive bool
	segments        []string
}

func Parse(raw string) (Pattern, error) {
	var p Pattern
	if err := source.ValidatePath(raw); err != nil {
		return p, err
	}
	p.Path = path.Clean(raw)
	if raw == "" {
		p.Path = "."
	}
	parts := strings.Split(p.Path, "/")
	first := -1
	for i, s := range parts {
		if strings.ContainsAny(s, "?[]") || strings.Contains(s, "**") && s != "**" || strings.Contains(s, "*") && s != "**" && i != len(parts)-1 {
			return p, diagnostic.New("ARROW_USAGE", "unsupported glob; use filename * or directory **")
		}
		if strings.Contains(s, "*") && first < 0 {
			first = i
		}
		if s == "**" {
			p.Recursive = true
		}
	}
	p.Glob = first >= 0
	if p.Glob {
		p.Base = strings.Join(parts[:first], "/")
		if p.Base == "" {
			p.Base = "."
		}
		p.segments = parts[first:]
	} else {
		p.Base = p.Path
	}
	return p, nil
}
func match(pattern, parts []string) bool {
	// Dynamic programming keeps repeated ** segments from causing exponential work.
	previous := make([]bool, len(parts)+1)
	previous[0] = true
	for _, segment := range pattern {
		next := make([]bool, len(parts)+1)
		for j := range next {
			if segment == "**" {
				next[j] = previous[j] || j > 0 && next[j-1]
			} else if j > 0 && previous[j-1] {
				ok, _ := path.Match(segment, parts[j-1])
				next[j] = ok
			}
		}
		previous = next
	}
	return previous[len(parts)]
}
func (p Pattern) Select(entries []Entry) []string {
	result := []string{}
	prefix := p.Base + "/"
	if p.Base == "." {
		prefix = ""
	}
	for _, e := range entries {
		if e.Kind != File || !strings.HasPrefix(e.Path, prefix) {
			continue
		}
		relative := strings.TrimPrefix(e.Path, prefix)
		if !p.Glob || match(p.segments, strings.Split(relative, "/")) {
			result = append(result, relative)
		}
	}
	sort.Strings(result)
	return result
}
func Escape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp) {
			q := strconv.QuoteRune(r)
			b.WriteString(q[1 : len(q)-1])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
