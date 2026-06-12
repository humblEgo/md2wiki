// Package config parses the declarative md2wiki.yaml configuration file.
package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// File is the full structure of an md2wiki.yaml file.
type File struct {
	BaseURL     string    `yaml:"baseUrl"`
	Email       string    `yaml:"email"`
	LayoutMode  string    `yaml:"layoutMode"`
	MermaidMode string    `yaml:"mermaidMode"`
	Banner      *bool     `yaml:"banner"` // nil means unset (default applies); the CLI resolves the default to true.
	Mappings    []Mapping `yaml:"mappings"`
}

// Mapping maps a single directory to a space and parent page.
type Mapping struct {
	Source      string `yaml:"source"`
	Space       string `yaml:"space"`
	RootPage    string `yaml:"rootPage"`
	LayoutMode  string `yaml:"layoutMode"`
	MermaidMode string `yaml:"mermaidMode"`
	Banner      *bool  `yaml:"banner"` // nil inherits the global value.
}

// Load reads the YAML at path and unmarshals it into a File. A missing file,
// malformed YAML, or an unknown (typo'd) key is returned as an error.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %q: %w", path, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("config: %q is empty", path)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // Reject unknown keys (e.g. a "layoutmode" typo) as errors.
	var f File
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}
	return &f, nil
}
