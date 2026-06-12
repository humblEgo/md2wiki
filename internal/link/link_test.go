package link

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/humblEgo/md2wiki/internal/convert"
	"github.com/humblEgo/md2wiki/internal/title"
	"github.com/humblEgo/md2wiki/internal/tree"
	"github.com/humblEgo/md2wiki/internal/walker"
)

func TestResolveLink(t *testing.T) {
	tr := &tree.Tree{Index: map[string]*tree.Node{
		"api/foo.md":      {Title: "Foo"},
		"guide/README.md": {Title: "Guide Home"},
		"top.md":          {Title: "Top"},
	}}
	cases := []struct {
		name, currentDoc, href string
		wantInternal           bool
		wantTitle, wantHref    string
	}{
		{"same dir", "api/bar.md", "foo.md", true, "Foo", ""},
		{"parent path", "guide/index.md", "../api/foo.md", true, "Foo", ""},
		{"readme to folder", "top.md", "guide/README.md", true, "Guide Home", ""},
		{"fragment dropped", "api/bar.md", "foo.md#sec", true, "Foo", ""},
		{"external http", "top.md", "https://e.com", false, "", "https://e.com"},
		{"mailto", "top.md", "mailto:a@b.com", false, "", "mailto:a@b.com"},
		{"protocol relative", "top.md", "//cdn/x.md", false, "", "//cdn/x.md"},
		{"anchor only", "top.md", "#sec", false, "", "#sec"},
		{"non-md relative", "top.md", "./img.png", false, "", "./img.png"},
		{"broken md", "top.md", "missing.md", false, "", "missing.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewResolver(tr, tc.currentDoc).ResolveLink(tc.href)
			if got.Internal != tc.wantInternal {
				t.Fatalf("Internal = %v, want %v (got %+v)", got.Internal, tc.wantInternal, got)
			}
			if tc.wantInternal {
				if got.PageTitle != tc.wantTitle {
					t.Errorf("PageTitle = %q, want %q", got.PageTitle, tc.wantTitle)
				}
			} else if got.Href != tc.wantHref {
				t.Errorf("Href = %q, want %q", got.Href, tc.wantHref)
			}
		})
	}
}

func TestIntegration_InternalLinkAcrossDocs(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"a.md": "# Alpha\n\nsee [beta](b.md)\n",
		"b.md": "# Beta\n",
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tr, err := walker.Walk(root, walker.ModeReadmeBody)
	if err != nil {
		t.Fatal(err)
	}
	if err := title.Resolve(root, tr); err != nil {
		t.Fatal(err)
	}

	aNode, ok := tr.Index["a.md"]
	if !ok {
		t.Fatal("a.md not in index")
	}
	src, err := os.ReadFile(filepath.Join(root, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := convert.Document(src, convert.WithLinkResolver(NewResolver(tr, aNode.SourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<ac:link><ri:page ri:content-title="Beta"/><ac:link-body>beta</ac:link-body></ac:link>`) {
		t.Errorf("expected internal link to Beta, got:\n%s", out)
	}
}
