package title

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/humblEgo/md2wiki/internal/tree"
	"github.com/humblEgo/md2wiki/internal/walker"
)

func TestExtractH1(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
		ok   bool
	}{
		{"atx", "# Title\n\nbody", "Title", true},
		{"setext", "Title\n=====\n\nbody", "Title", true},
		{"only h2", "## Sub\n\nbody", "", false},
		{"none", "just text\n", "", false},
		{"hash in code fence", "```\n# Not a title\n```\n", "", false},
		{"frontmatter then h1", "---\ntitle: x\n---\n\n# Real\n", "Real", true},
		{"leading blank lines", "\n\n# Title\n", "Title", true},
		{"first of multiple", "# First\n\n# Second\n", "First", true},
		{"inline emphasis stripped", "# **Bold** title\n", "Bold title", true},
		{"trim whitespace", "#   Spaced   \n", "Spaced", true},
		{"h2 before h1 takes h1", "## Sub\n\n# Main\n", "Main", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := extractH1([]byte(tc.src))
			if got != tc.want || ok != tc.ok {
				t.Errorf("extractH1(%q) = (%q,%v), want (%q,%v)", tc.src, got, ok, tc.want, tc.ok)
			}
		})
	}
}

// writeTree materializes a (relative path -> content) map as files under a fresh
// temporary directory and returns that directory's path.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

// pageTitles collects the Title of every page node (KindPage) in depth-first
// pre-order, so the returned slice has a stable, predictable order for assertions.
func pageTitles(n *tree.Node) []string {
	var out []string
	var walk func(*tree.Node)
	walk = func(x *tree.Node) {
		if x.Kind == tree.KindPage {
			out = append(out, x.Title)
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

func childByName(t *testing.T, n *tree.Node, name string) *tree.Node {
	t.Helper()
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("child %q not found under %q", name, n.Name)
	return nil
}

func resolveTree(t *testing.T, files map[string]string) (*tree.Tree, string) {
	t.Helper()
	root := writeTree(t, files)
	tr, err := walker.Walk(root, walker.ModeReadmeBody)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if err := Resolve(root, tr); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return tr, root
}

func TestResolve_ConflictPathSuffix(t *testing.T) {
	tr, _ := resolveTree(t, map[string]string{
		"a/page.md": "# Same\n",
		"b/page.md": "# Same\n",
	})
	got := pageTitles(tr.Root)
	want := []string{"Same (a/page.md)", "Same (b/page.md)"}
	if !slices.Equal(got, want) {
		t.Errorf("page titles = %v, want %v", got, want)
	}
}

func TestResolve_NonCollidingStaysClean(t *testing.T) {
	tr, _ := resolveTree(t, map[string]string{
		"a/page.md": "# Alpha\n",
		"b/page.md": "# Beta\n",
	})
	got := pageTitles(tr.Root)
	want := []string{"Alpha", "Beta"}
	if !slices.Equal(got, want) {
		t.Errorf("page titles = %v, want %v", got, want)
	}
}

func TestResolve_ConflictTitleStableUnderGroupChange(t *testing.T) {
	titleOf := func(tr *tree.Tree, relPath string) string {
		var found string
		var walk func(*tree.Node)
		walk = func(n *tree.Node) {
			if n.RelPath == relPath {
				found = n.Title
			}
			for _, c := range n.Children {
				walk(c)
			}
		}
		walk(tr.Root)
		return found
	}
	tr2, _ := resolveTree(t, map[string]string{
		"a/page.md": "# Same\n",
		"b/page.md": "# Same\n",
	})
	tr3, _ := resolveTree(t, map[string]string{
		"a/page.md": "# Same\n",
		"b/page.md": "# Same\n",
		"c/page.md": "# Same\n",
	})
	if got2, got3 := titleOf(tr2, "b/page.md"), titleOf(tr3, "b/page.md"); got2 != got3 {
		t.Errorf("b/page.md title changed when collision group grew: %q vs %q", got2, got3)
	}
}

func TestResolve_FallbackToName(t *testing.T) {
	tr, _ := resolveTree(t, map[string]string{"notitle.md": "no heading here\n"})
	if got := pageTitles(tr.Root); !slices.Equal(got, []string{"notitle"}) {
		t.Errorf("page titles = %v, want [notitle]", got)
	}
}

func TestResolve_FolderTitleFromReadme(t *testing.T) {
	tr, _ := resolveTree(t, map[string]string{
		"sub/README.md": "# Sub Home\n",
		"sub/page.md":   "# Page\n",
	})
	sub := childByName(t, tr.Root, "sub")
	if sub.Title != "Sub Home" {
		t.Errorf("sub title = %q, want %q", sub.Title, "Sub Home")
	}
}

func TestResolve_Deterministic(t *testing.T) {
	files := map[string]string{"a/p.md": "# T\n", "b/p.md": "# T\n"}
	root := writeTree(t, files)
	tr1, err := walker.Walk(root, walker.ModeReadmeBody)
	if err != nil {
		t.Fatal(err)
	}
	if err := Resolve(root, tr1); err != nil {
		t.Fatal(err)
	}
	tr2, err := walker.Walk(root, walker.ModeReadmeBody)
	if err != nil {
		t.Fatal(err)
	}
	if err := Resolve(root, tr2); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(pageTitles(tr1.Root), pageTitles(tr2.Root)) {
		t.Errorf("non-deterministic: %v vs %v", pageTitles(tr1.Root), pageTitles(tr2.Root))
	}
}

func TestResolve_ReadError(t *testing.T) {
	root := writeTree(t, map[string]string{"p.md": "# T\n"})
	tr, err := walker.Walk(root, walker.ModeReadmeBody)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "p.md")); err != nil {
		t.Fatal(err)
	}
	if err := Resolve(root, tr); err == nil {
		t.Fatal("expected error when a SourcePath file is missing")
	}
}
