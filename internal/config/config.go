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
	BaseURL     string    `yaml:"baseUrl,omitempty"`
	Email       string    `yaml:"email,omitempty"`
	LayoutMode  string    `yaml:"layoutMode,omitempty"`
	MermaidMode string    `yaml:"mermaidMode,omitempty"`
	Banner      *bool     `yaml:"banner,omitempty"` // nil means unset (default applies); the CLI resolves the default to true.
	Mappings    []Mapping `yaml:"mappings,omitempty"`
}

// Mapping maps a single directory to a space and parent page.
type Mapping struct {
	Source      string `yaml:"source,omitempty"`
	Space       string `yaml:"space,omitempty"`
	RootPage    string `yaml:"rootPage,omitempty"`
	LayoutMode  string `yaml:"layoutMode,omitempty"`
	MermaidMode string `yaml:"mermaidMode,omitempty"`
	Banner      *bool  `yaml:"banner,omitempty"` // nil inherits the global value.
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

// Marshal serializes the File back to md2wiki.yaml bytes. Empty optional fields are
// omitted (omitempty tags) so generated files stay clean. A non-nil *false Banner is
// still emitted, because only nil pointers count as empty.
func (f *File) Marshal() ([]byte, error) {
	data, err := yaml.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("config: marshal: %w", err)
	}
	return data, nil
}
