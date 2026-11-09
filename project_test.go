package project

import "testing"

func TestEmbeddedVersion(t *testing.T) {
	if v := Version(); v == "" || v == "unknown" {
		t.Fatal("main.project.yaml must contain a version")
	}
}
func TestInvalidVersionMetadata(t *testing.T) {
	for _, data := range []string{"version: [", "{}", "version: ''"} {
		if decodeVersion([]byte(data)) != "unknown" {
			t.Errorf("accepted invalid version metadata: %q", data)
		}
	}
}
