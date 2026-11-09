// Package source parses explicit identities without inferring repositories.
package source

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/flarebyte/gh-arrow/internal/diagnostic"
)

type Source struct{ Scheme, Repository, Alias, Path string }

var ownerRE = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)
var repoRE = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}$`)

func ValidOwner(s string) bool { return ownerRE.MatchString(s) && !strings.Contains(s, "--") }
func ValidRepository(owner, repo string) bool {
	return ValidOwner(owner) && repoRE.MatchString(repo) && repo != "." && repo != ".."
}
func ValidAlias(s string) bool { return s != "" && !strings.ContainsAny(s, ":/\\\r\n\x00") }
func ValidatePath(p string) error {
	if !utf8.ValidString(p) || strings.HasPrefix(p, "/") || strings.ContainsAny(p, ":\\\x00") {
		return diagnostic.New("ARROW_USAGE", "invalid repository-relative path")
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." {
			return diagnostic.New("ARROW_USAGE", "parent traversal is not allowed")
		}
	}
	return nil
}
func Parse(raw string) (Source, error) {
	p := strings.Split(raw, ":")
	var s Source
	switch {
	case len(p) == 4 && p[0] == "github" && ValidRepository(p[1], p[2]):
		s = Source{Scheme: "github", Repository: p[1] + "/" + p[2], Path: p[3]}
	case len(p) == 3 && p[0] == "local" && ValidAlias(p[1]):
		s = Source{Scheme: "local", Alias: p[1], Path: p[2]}
	default:
		return s, diagnostic.New("ARROW_USAGE", "expected github:<owner>:<repo>:<path> or local:<alias>:<path>")
	}
	return s, ValidatePath(s.Path)
}
