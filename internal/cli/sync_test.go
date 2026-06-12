package cli

import (
	"strings"
	"testing"

	"github.com/humblEgo/md2wiki/internal/convert"
	"github.com/humblEgo/md2wiki/internal/walker"
)

func TestParseLayout(t *testing.T) {
	cases := []struct {
		in   string
		want walker.Mode
		ok   bool
	}{
		{"readme-body", walker.ModeReadmeBody, true},
		{"mirror", walker.ModeMirror, true},
		{"bogus", 0, false},
	}
	for _, c := range cases {
		got, err := parseLayout(c.in)
		if (err == nil) != c.ok {
			t.Errorf("parseLayout(%q) err=%v, want ok=%v", c.in, err, c.ok)
		}
		if c.ok && got != c.want {
			t.Errorf("parseLayout(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseMermaidMode(t *testing.T) {
	cases := []struct {
		in   string
		want convert.MermaidMode
		ok   bool
	}{
		{"details", convert.MermaidDetails, true},
		{"render", convert.MermaidRender, true},
		{"raw", convert.MermaidRaw, true},
		{"bogus", 0, false},
	}
	for _, c := range cases {
		got, err := parseMermaidMode(c.in)
		if (err == nil) != c.ok {
			t.Errorf("parseMermaidMode(%q) err=%v, want ok=%v", c.in, err, c.ok)
		}
		if c.ok && got != c.want {
			t.Errorf("parseMermaidMode(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestLoadConfig_Valid(t *testing.T) {
	v := newConfigViper()
	v.Set("base-url", "https://x.atlassian.net")
	v.Set("email", "a@b.com")
	v.Set("space", "DOCS")
	v.Set("api-token", "tok")
	v.Set("layout-mode", "mirror")
	v.Set("mermaid-mode", "raw")
	cfg, err := loadConfig(v, "/some/dir")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.dir != "/some/dir" || cfg.baseURL != "https://x.atlassian.net" || cfg.email != "a@b.com" ||
		cfg.space != "DOCS" || cfg.token != "tok" {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.layout != walker.ModeMirror {
		t.Errorf("layout = %v, want mirror", cfg.layout)
	}
	if cfg.mermaidMode != convert.MermaidRaw {
		t.Errorf("mermaidMode = %v, want raw", cfg.mermaidMode)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	v := newConfigViper()
	v.Set("base-url", "u")
	v.Set("email", "e")
	v.Set("space", "s")
	v.Set("api-token", "t")
	cfg, err := loadConfig(v, "/d")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.layout != walker.ModeReadmeBody {
		t.Errorf("default layout = %v, want readme-body", cfg.layout)
	}
	if cfg.mermaidMode != convert.MermaidDetails {
		t.Errorf("default mermaidMode = %v, want details", cfg.mermaidMode)
	}
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	v := newConfigViper()
	v.Set("email", "a@b.com")
	_, err := loadConfig(v, "/d")
	if err == nil {
		t.Fatal("expected error for missing required config")
	}
	for _, want := range []string{"--base-url", "--space", "MD2WIKI_API_TOKEN"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err.Error(), want)
		}
	}
}

func TestLoadConfig_BadLayout(t *testing.T) {
	v := newConfigViper()
	v.Set("base-url", "u")
	v.Set("email", "e")
	v.Set("space", "s")
	v.Set("api-token", "t")
	v.Set("layout-mode", "nope")
	if _, err := loadConfig(v, "/d"); err == nil {
		t.Fatal("expected error for unknown layout-mode")
	}
}

func TestLoadConfig_BadMermaidMode(t *testing.T) {
	v := newConfigViper()
	v.Set("base-url", "u")
	v.Set("email", "e")
	v.Set("space", "s")
	v.Set("api-token", "t")
	v.Set("mermaid-mode", "nope")
	if _, err := loadConfig(v, "/d"); err == nil {
		t.Fatal("expected error for unknown mermaid-mode")
	}
}

func TestLoadConfig_TokenFromEnv(t *testing.T) {
	t.Setenv("MD2WIKI_API_TOKEN", "envtok")
	v := newConfigViper()
	v.Set("base-url", "u")
	v.Set("email", "e")
	v.Set("space", "s")
	cfg, err := loadConfig(v, "/d")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.token != "envtok" {
		t.Errorf("token = %q, want envtok (from env)", cfg.token)
	}
}

func TestSyncCommandRegistered(t *testing.T) {
	root := NewRootCmd()
	sub, _, err := root.Find([]string{"sync"})
	if err != nil {
		t.Fatalf("Find(sync): %v", err)
	}
	if sub == nil || sub.Name() != "sync" {
		t.Fatalf("sync subcommand not registered, got %v", sub)
	}
}

func TestApplyCommandRegistered(t *testing.T) {
	root := NewRootCmd()
	sub, _, err := root.Find([]string{"apply"})
	if err != nil {
		t.Fatalf("Find(apply): %v", err)
	}
	if sub == nil || sub.Name() != "apply" {
		t.Fatalf("apply subcommand not registered, got %v", sub)
	}
}

func TestLoadConfig_ParentIDOptional(t *testing.T) {
	v := newConfigViper()
	v.Set("base-url", "u")
	v.Set("email", "e")
	v.Set("space", "s")
	v.Set("api-token", "t")

	cfg, err := loadConfig(v, "/d")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.parentID != "" {
		t.Errorf("default parentID = %q, want empty", cfg.parentID)
	}

	v.Set("parent-id", "123456")
	cfg2, err := loadConfig(v, "/d")
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.parentID != "123456" {
		t.Errorf("parentID = %q, want 123456", cfg2.parentID)
	}
}

func TestLoadConfig_BannerDefaultOn(t *testing.T) {
	v := newConfigViper()
	v.Set("base-url", "u")
	v.Set("email", "e")
	v.Set("space", "s")
	v.Set("api-token", "t")
	cfg, err := loadConfig(v, "/d")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.banner {
		t.Errorf("default banner = %v, want true", cfg.banner)
	}
}

func TestLoadConfig_BannerOff(t *testing.T) {
	v := newConfigViper()
	v.Set("base-url", "u")
	v.Set("email", "e")
	v.Set("space", "s")
	v.Set("api-token", "t")
	v.Set("banner", false)
	cfg, err := loadConfig(v, "/d")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.banner {
		t.Errorf("banner = %v, want false", cfg.banner)
	}
}
