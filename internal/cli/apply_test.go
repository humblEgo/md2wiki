package cli

import (
	"path/filepath"
	"testing"

	"github.com/humblEgo/md2wiki/internal/config"
	"github.com/humblEgo/md2wiki/internal/convert"
	"github.com/humblEgo/md2wiki/internal/walker"
)

func baseFile() *config.File {
	return &config.File{
		BaseURL: "https://file.atlassian.net",
		Email:   "file@b.com",
		Mappings: []config.Mapping{
			{Source: "docs/specs", Space: "PROD", RootPage: "123"},
		},
	}
}

// Flag/env values (via viper) override the file's global values.
func TestResolveApply_FlagOverridesFile(t *testing.T) {
	v := newApplyViper()
	v.Set("base-url", "https://flag.atlassian.net")
	v.Set("api-token", "tok")
	plan, err := resolveApply(baseFile(), v, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if plan.baseURL != "https://flag.atlassian.net" {
		t.Errorf("baseURL = %q, want flag value", plan.baseURL)
	}
	if plan.email != "file@b.com" {
		t.Errorf("email = %q, want file value", plan.email)
	}
}

// The file's global values alone are sufficient (no flag/env, only the token comes from env).
func TestResolveApply_FileOnly(t *testing.T) {
	v := newApplyViper()
	v.Set("api-token", "tok")
	plan, err := resolveApply(baseFile(), v, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if plan.baseURL != "https://file.atlassian.net" || plan.email != "file@b.com" {
		t.Errorf("conn = %q/%q", plan.baseURL, plan.email)
	}
}

func TestResolveApply_MissingRequired(t *testing.T) {
	v := newApplyViper() // no token set
	if _, err := resolveApply(baseFile(), v, "/repo"); err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestResolveApply_NoMappings(t *testing.T) {
	v := newApplyViper()
	v.Set("api-token", "tok")
	f := baseFile()
	f.Mappings = nil
	if _, err := resolveApply(f, v, "/repo"); err == nil {
		t.Fatal("expected error for empty mappings")
	}
}

// A per-mapping layoutMode overrides the global/default value, and source is normalized to an absolute path relative to the config directory.
func TestResolveApply_MappingOverrideAndPath(t *testing.T) {
	v := newApplyViper()
	v.Set("api-token", "tok")
	f := baseFile()
	f.LayoutMode = "readme-body"
	f.Mappings = []config.Mapping{
		{Source: "docs/specs", Space: "PROD"},                    // inherits global -> readme-body
		{Source: "docs/ops", Space: "OPS", LayoutMode: "mirror"}, // override
		{Source: "/abs/path", Space: "ABS"},                      // absolute path used as-is
	}
	plan, err := resolveApply(f, v, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if plan.mappings[0].layout != walker.ModeReadmeBody {
		t.Errorf("m0 layout = %v, want readme-body", plan.mappings[0].layout)
	}
	if plan.mappings[1].layout != walker.ModeMirror {
		t.Errorf("m1 layout = %v, want mirror", plan.mappings[1].layout)
	}
	if plan.mappings[0].source != filepath.Join("/repo", "docs/specs") {
		t.Errorf("m0 source = %q, want joined", plan.mappings[0].source)
	}
	if plan.mappings[2].source != "/abs/path" {
		t.Errorf("abs source = %q, want unchanged", plan.mappings[2].source)
	}
	if plan.mappings[0].mermaid != convert.MermaidDetails {
		t.Errorf("default mermaid = %v, want details", plan.mappings[0].mermaid)
	}
}

func TestResolveApply_MappingMissingSpace(t *testing.T) {
	v := newApplyViper()
	v.Set("api-token", "tok")
	f := baseFile()
	f.Mappings = []config.Mapping{{Source: "docs/specs"}} // missing space
	if _, err := resolveApply(f, v, "/repo"); err == nil {
		t.Fatal("expected error for mapping missing space")
	}
}

func TestResolveApply_BadLayout(t *testing.T) {
	v := newApplyViper()
	v.Set("api-token", "tok")
	f := baseFile()
	f.Mappings[0].LayoutMode = "bogus"
	if _, err := resolveApply(f, v, "/repo"); err == nil {
		t.Fatal("expected error for bad layout-mode")
	}
}

func boolPtr(b bool) *bool { return &b }

func TestResolveApply_BannerResolution(t *testing.T) {
	v := newApplyViper()
	v.Set("api-token", "tok")
	f := baseFile()
	f.Banner = boolPtr(false) // disable banner globally
	f.Mappings = []config.Mapping{
		{Source: "a", Space: "A"},                        // inherits global -> false
		{Source: "b", Space: "B", Banner: boolPtr(true)}, // per-mapping override -> true
	}
	plan, err := resolveApply(f, v, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if plan.mappings[0].banner != false {
		t.Errorf("m0 banner = %v, want false (inherit global)", plan.mappings[0].banner)
	}
	if plan.mappings[1].banner != true {
		t.Errorf("m1 banner = %v, want true (override)", plan.mappings[1].banner)
	}
}

func TestResolveApply_BannerDefaultsTrue(t *testing.T) {
	v := newApplyViper()
	v.Set("api-token", "tok")
	f := baseFile() // Banner unset globally, and unset on the mapping too
	plan, err := resolveApply(f, v, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.mappings[0].banner {
		t.Errorf("default banner = %v, want true", plan.mappings[0].banner)
	}
}
