package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_Valid(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "md2wiki.yaml", `
baseUrl: https://x.atlassian.net
email: a@b.com
layoutMode: readme-body
mermaidMode: details
mappings:
  - source: docs/specs
    space: PROD
    rootPage: "123"
  - source: docs/runbooks
    space: OPS
    layoutMode: mirror
`)
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if f.BaseURL != "https://x.atlassian.net" || f.Email != "a@b.com" {
		t.Errorf("global = %q/%q", f.BaseURL, f.Email)
	}
	if len(f.Mappings) != 2 {
		t.Fatalf("mappings = %d, want 2", len(f.Mappings))
	}
	if f.Mappings[0].Source != "docs/specs" || f.Mappings[0].Space != "PROD" || f.Mappings[0].RootPage != "123" {
		t.Errorf("m0 = %+v", f.Mappings[0])
	}
	if f.Mappings[1].LayoutMode != "mirror" {
		t.Errorf("m1 layout = %q, want mirror", f.Mappings[1].LayoutMode)
	}
	if f.MermaidMode != "details" {
		t.Errorf("global mermaidMode = %q, want details", f.MermaidMode)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_BrokenYAML(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "md2wiki.yaml", "mappings: [oops")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for broken yaml")
	}
}

func TestLoad_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "md2wiki.yaml", "baseurl: typo\nmappings: []\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for unknown key (typo)")
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "md2wiki.yaml", "   \n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestLoad_Banner(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "md2wiki.yaml", `
banner: false
mappings:
  - source: docs
    space: PROD
    banner: true
  - source: ops
    space: OPS
`)
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if f.Banner == nil || *f.Banner != false {
		t.Errorf("global banner = %v, want *false", f.Banner)
	}
	if f.Mappings[0].Banner == nil || *f.Mappings[0].Banner != true {
		t.Errorf("m0 banner = %v, want *true", f.Mappings[0].Banner)
	}
	if f.Mappings[1].Banner != nil {
		t.Errorf("m1 banner = %v, want nil (unset)", f.Mappings[1].Banner)
	}
}

func TestMarshal_RoundTrip(t *testing.T) {
	tr := true
	in := &File{
		BaseURL:     "https://x.atlassian.net",
		Email:       "a@b.com",
		LayoutMode:  "readme-body",
		MermaidMode: "details",
		Banner:      &tr,
		Mappings: []Mapping{
			{Source: "docs/specs", Space: "PROD", RootPage: "123"},
			{Source: "docs/ops", Space: "OPS", LayoutMode: "mirror"},
		},
	}
	data, err := in.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	p := writeFile(t, dir, "md2wiki.yaml", string(data))
	out, err := Load(p)
	if err != nil {
		t.Fatalf("round-trip Load failed: %v\n--- yaml ---\n%s", err, data)
	}
	if out.BaseURL != in.BaseURL || out.Email != in.Email || out.LayoutMode != in.LayoutMode {
		t.Errorf("globals mismatch: %+v", out)
	}
	if len(out.Mappings) != 2 || out.Mappings[1].LayoutMode != "mirror" {
		t.Errorf("mappings mismatch: %+v", out.Mappings)
	}
	if out.Banner == nil || *out.Banner != true {
		t.Errorf("banner = %v, want *true", out.Banner)
	}
}

func TestMarshal_OmitsEmpty(t *testing.T) {
	in := &File{Mappings: []Mapping{{Source: "docs", Space: "DOCS"}}}
	data, err := in.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, bad := range []string{"baseUrl:", "email:", "layoutMode:", "mermaidMode:", "banner:", "rootPage:"} {
		if strings.Contains(s, bad) {
			t.Errorf("expected %q to be omitted, got:\n%s", bad, s)
		}
	}
}
