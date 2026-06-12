package walker

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/humblEgo/md2wiki/internal/tree"
)

// writeTree materializes a map of (relative path -> file content) into a fresh
// temporary directory and returns that directory's path. Any missing parent
// directories are created automatically.
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

func childNames(n *tree.Node) []string {
	names := make([]string, 0, len(n.Children))
	for _, c := range n.Children {
		names = append(names, c.Name)
	}
	return names
}

func childByName(t *testing.T, n *tree.Node, name string) *tree.Node {
	t.Helper()
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("child %q not found under %q (have %v)", name, n.Name, childNames(n))
	return nil
}

func mustWalk(t *testing.T, root string, mode Mode) *tree.Tree {
	t.Helper()
	tr, err := Walk(root, mode)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	return tr
}

func TestWalk_FlatFiles_ReadmeBody(t *testing.T) {
	root := writeTree(t, map[string]string{"a.md": "a", "b.md": "b", "c.md": "c"})
	tr := mustWalk(t, root, ModeReadmeBody)

	if tr.Root.Kind != tree.KindFolder {
		t.Errorf("root kind = %v, want folder", tr.Root.Kind)
	}
	if tr.Root.RelPath != "." {
		t.Errorf("root relpath = %q, want %q", tr.Root.RelPath, ".")
	}
	if got, want := childNames(tr.Root), []string{"a", "b", "c"}; !slices.Equal(got, want) {
		t.Errorf("children = %v, want %v", got, want)
	}
	for _, name := range []string{"a.md", "b.md", "c.md"} {
		n, ok := tr.Index[name]
		if !ok {
			t.Errorf("index missing %q", name)
			continue
		}
		if n.Kind != tree.KindPage || n.SourcePath != name {
			t.Errorf("index[%q] = %+v, want page with sourcepath %q", name, n, name)
		}
	}
}

func TestWalk_ReadmeAbsorbed_ReadmeBody(t *testing.T) {
	root := writeTree(t, map[string]string{"README.md": "home", "a.md": "a"})
	tr := mustWalk(t, root, ModeReadmeBody)

	if tr.Root.SourcePath != "README.md" {
		t.Errorf("root sourcepath = %q, want %q", tr.Root.SourcePath, "README.md")
	}
	if got, want := childNames(tr.Root), []string{"a"}; !slices.Equal(got, want) {
		t.Errorf("children = %v, want %v (README must not be a child)", got, want)
	}
	if tr.Index["README.md"] != tr.Root {
		t.Errorf("index[README.md] should map to root folder node")
	}
}

func TestWalk_Nested_ReadmeBody(t *testing.T) {
	root := writeTree(t, map[string]string{
		"README.md":     "home",
		"sub/README.md": "subhome",
		"sub/page.md":   "p",
	})
	tr := mustWalk(t, root, ModeReadmeBody)

	if got, want := childNames(tr.Root), []string{"sub"}; !slices.Equal(got, want) {
		t.Fatalf("root children = %v, want %v", got, want)
	}
	sub := childByName(t, tr.Root, "sub")
	if sub.Kind != tree.KindFolder {
		t.Errorf("sub kind = %v, want folder", sub.Kind)
	}
	if sub.SourcePath != "sub/README.md" {
		t.Errorf("sub sourcepath = %q, want %q", sub.SourcePath, "sub/README.md")
	}
	if got, want := childNames(sub), []string{"page"}; !slices.Equal(got, want) {
		t.Errorf("sub children = %v, want %v", got, want)
	}
	if tr.Index["sub/README.md"] != sub {
		t.Errorf("index[sub/README.md] should map to sub folder")
	}
	if p := tr.Index["sub/page.md"]; p == nil || p.Kind != tree.KindPage {
		t.Errorf("index[sub/page.md] should be a page, got %+v", p)
	}
}

func TestWalk_FolderWithoutReadme_ReadmeBody(t *testing.T) {
	root := writeTree(t, map[string]string{"sub/page.md": "p"})
	tr := mustWalk(t, root, ModeReadmeBody)

	if tr.Root.SourcePath != "" {
		t.Errorf("root sourcepath = %q, want empty", tr.Root.SourcePath)
	}
	sub := childByName(t, tr.Root, "sub")
	if sub.SourcePath != "" {
		t.Errorf("sub sourcepath = %q, want empty", sub.SourcePath)
	}
	if got, want := childNames(sub), []string{"page"}; !slices.Equal(got, want) {
		t.Errorf("sub children = %v, want %v", got, want)
	}
}

func TestWalk_PrunesDirsWithoutMarkdown(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.md":           "a",
		"nomd/notes.txt": "x",
	})
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr := mustWalk(t, root, ModeReadmeBody)

	if got, want := childNames(tr.Root), []string{"a"}; !slices.Equal(got, want) {
		t.Errorf("children = %v, want %v (empty/nomd dirs pruned)", got, want)
	}
}

func TestWalk_IgnoresDotfilesAndNonMarkdown(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.md":        "a",
		".hidden.md":  "h",
		".git/config": "c",
		"notes.txt":   "n",
	})
	tr := mustWalk(t, root, ModeReadmeBody)

	if got, want := childNames(tr.Root), []string{"a"}; !slices.Equal(got, want) {
		t.Errorf("children = %v, want %v", got, want)
	}
	if _, ok := tr.Index[".hidden.md"]; ok {
		t.Errorf("hidden .md must not be indexed")
	}
}

func TestWalk_ChildrenSortedDeterministically(t *testing.T) {
	root := writeTree(t, map[string]string{"z.md": "z", "a.md": "a", "m.md": "m"})
	tr := mustWalk(t, root, ModeReadmeBody)
	if got, want := childNames(tr.Root), []string{"a", "m", "z"}; !slices.Equal(got, want) {
		t.Errorf("children = %v, want sorted %v", got, want)
	}
}

func TestWalk_EmptyRootReturnsEmptyFolder(t *testing.T) {
	root := t.TempDir()
	tr := mustWalk(t, root, ModeReadmeBody)
	if tr.Root == nil {
		t.Fatal("root should never be nil")
	}
	if tr.Root.Kind != tree.KindFolder || tr.Root.RelPath != "." {
		t.Errorf("root = %+v, want empty folder with relpath .", tr.Root)
	}
	if len(tr.Root.Children) != 0 {
		t.Errorf("root children = %v, want none", childNames(tr.Root))
	}
}

func TestWalk_MirrorMode_ReadmeIsChild(t *testing.T) {
	root := writeTree(t, map[string]string{"README.md": "home", "a.md": "a"})
	tr := mustWalk(t, root, ModeMirror)

	if tr.Root.SourcePath != "" {
		t.Errorf("root sourcepath = %q, want empty in mirror mode", tr.Root.SourcePath)
	}
	// Children are sorted by byte order, so 'R' (82) sorts before 'a' (97)
	// and README therefore comes first.
	if got, want := childNames(tr.Root), []string{"README", "a"}; !slices.Equal(got, want) {
		t.Errorf("children = %v, want %v (README is a page)", got, want)
	}
	readme := childByName(t, tr.Root, "README")
	if readme.Kind != tree.KindPage || readme.SourcePath != "README.md" {
		t.Errorf("README node = %+v, want page with sourcepath README.md", readme)
	}
	if tr.Index["README.md"] != readme {
		t.Errorf("index[README.md] should map to the README page (not folder) in mirror mode")
	}
}

func TestWalk_MirrorMode_NestedReadme(t *testing.T) {
	root := writeTree(t, map[string]string{"sub/README.md": "s", "sub/page.md": "p"})
	tr := mustWalk(t, root, ModeMirror)

	sub := childByName(t, tr.Root, "sub")
	if sub.SourcePath != "" {
		t.Errorf("sub sourcepath = %q, want empty in mirror mode", sub.SourcePath)
	}
	if got, want := childNames(sub), []string{"README", "page"}; !slices.Equal(got, want) {
		t.Errorf("sub children = %v, want %v", got, want)
	}
}

func TestWalk_Errors(t *testing.T) {
	t.Run("nonexistent root", func(t *testing.T) {
		_, err := Walk(filepath.Join(t.TempDir(), "nope"), ModeReadmeBody)
		if err == nil {
			t.Fatal("expected error for nonexistent root")
		}
	})
	t.Run("file as root", func(t *testing.T) {
		root := writeTree(t, map[string]string{"a.md": "a"})
		_, err := Walk(filepath.Join(root, "a.md"), ModeReadmeBody)
		if err == nil {
			t.Fatal("expected error when root is a file")
		}
	})
}

func TestWalk_IgnoresSymlinks(t *testing.T) {
	root := writeTree(t, map[string]string{
		"real.md":         "r",
		"target/inner.md": "i",
	})
	// Create a symlink file pointing at real.md and a symlink directory
	// pointing at target/, to verify that both kinds of symlink are skipped.
	if err := os.Symlink(filepath.Join(root, "real.md"), filepath.Join(root, "link.md")); err != nil {
		t.Skipf("symlink unsupported on this platform: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "target"), filepath.Join(root, "linkdir")); err != nil {
		t.Skipf("symlink unsupported on this platform: %v", err)
	}

	tr := mustWalk(t, root, ModeReadmeBody)

	// Symlinks are not followed, so neither link.md (the page) nor linkdir
	// (the folder) should appear anywhere in the tree.
	for _, name := range childNames(tr.Root) {
		if name == "link" || name == "linkdir" {
			t.Errorf("symlink %q must be excluded; children = %v", name, childNames(tr.Root))
		}
	}
	if _, ok := tr.Index["link.md"]; ok {
		t.Errorf("symlinked .md must not be indexed")
	}
	// Control check: the real (non-symlinked) file is still included as expected.
	if _, ok := tr.Index["real.md"]; !ok {
		t.Errorf("real.md should be present")
	}
}
