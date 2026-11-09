// Package project exposes embedded build metadata for the CLI, not a library API.
package project

import (
	_ "embed"

	"go.yaml.in/yaml/v3"
)

//go:embed main.project.yaml
var metadata []byte

// Version reads the same committed version source used by release preflight.
// Embedding avoids runtime files and dependency-specific linker flag injection.
func Version() string { return decodeVersion(metadata) }
func decodeVersion(data []byte) string {
	var p struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &p); err != nil || p.Version == "" {
		return "unknown"
	}
	return p.Version
}
