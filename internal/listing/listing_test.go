package listing

import (
	"reflect"
	"testing"
)

func TestPatternsA07(t *testing.T) {
	files := []Entry{{Path: "docs/a.md", Kind: File}, {Path: "docs/sub/b.md", Kind: File}, {Path: "docs/sub/deep/c.txt", Kind: File}}
	for _, tt := range []struct {
		pattern string
		want    []string
	}{
		{"docs/*.md", []string{"a.md"}}, {"docs/**/*.md", []string{"a.md", "sub/b.md"}}, {"docs/**", []string{"a.md", "sub/b.md", "sub/deep/c.txt"}}, {"docs/**/sub/*.md", []string{"sub/b.md"}}, {"docs/no*.md", []string{}},
	} {
		t.Run(tt.pattern, func(t *testing.T) {
			p, err := Parse(tt.pattern)
			if err != nil {
				t.Fatal(err)
			}
			got := p.Select(files)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("%v != %v", got, tt.want)
			}
		})
	}
	for _, s := range []string{"a?", "[a]", "a**b", "d*/x", "../x"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
}

func TestOutputA13A18(t *testing.T) {
	if got := Escape("a\nb\x1b[31m"); got != "a\\nb\\x1b[31m" {
		t.Fatalf("%q", got)
	}
	p, _ := Parse("docs/*.md")
	got := p.Select([]Entry{{Path: "docs/z.md", Kind: File}, {Path: "docs/a.md", Kind: File}, {Path: "docs/s.md", Kind: Symlink}})
	if !reflect.DeepEqual(got, []string{"a.md", "z.md"}) {
		t.Fatal(got)
	}
}

func FuzzPattern(f *testing.F) {
	for _, s := range []string{"docs/**/a*.md", "**", "a?"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		p, e := Parse(s)
		if e == nil {
			p.Select([]Entry{{Path: "a/b.md", Kind: File}})
		}
	})
}
