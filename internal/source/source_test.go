package source

import "testing"

func TestParseA04(t *testing.T) {
	for _, raw := range []string{"ftp:host:file", "github::r:x", "github:o::x", "local::x", "github:o:r:../secret", "local:a:/etc/passwd", `local:a:C:\secret`, "github:o:r:a:b", "github:bad/name:r:x", "local:a:a/../../b", "local:a:a\x00b"} {
		if _, err := Parse(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	got, err := Parse("github:Owner:repo:docs/read me.md")
	if err != nil || got.Repository != "Owner/repo" || got.Path != "docs/read me.md" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := Parse("local:project:"); err != nil {
		t.Fatal(err)
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"github:o:r:docs/a", "local:a:", "local:a:../x", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		s, err := Parse(raw)
		if err == nil {
			if err := ValidatePath(s.Path); err != nil {
				t.Fatal(err)
			}
		}
	})
}
