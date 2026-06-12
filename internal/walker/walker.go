// Package walker traverses a Markdown directory and builds a document tree
// that mirrors the on-disk layout. The resulting tree is what the rest of the
// pipeline mirrors into Confluence, with the repository as the single source
// of truth.
package walker

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/humblEgo/md2wiki/internal/tree"
)

// Mode determines how a directory and its README.md are structured in the
// resulting document tree.
type Mode int

const (
	// ModeReadmeBody absorbs a directory's README.md as the body of that
	// folder's page (the default). The README does not become a separate
	// child page; instead the folder page itself carries the README content.
	ModeReadmeBody Mode = iota
	// ModeMirror mirrors the filesystem one-to-one. README.md is treated like
	// any other Markdown file and becomes an ordinary child page rather than
	// being absorbed into the folder page.
	ModeMirror
)

const readmeName = "README.md"

// Walk traverses the root directory and builds a document tree according to
// mode. It returns an error if root does not exist or is not a directory.
func Walk(root string, mode Mode) (*tree.Tree, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("walker: cannot access root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("walker: root %q is not a directory", root)
	}

	tr := &tree.Tree{Index: make(map[string]*tree.Node)}
	node, err := buildFolder(root, root, mode, tr)
	if err != nil {
		return nil, err
	}
	if node == nil {
		// Even when the tree under root contains no .md files at all,
		// buildFolder prunes it to nil; here we still return an empty root
		// folder node so callers always get a usable, non-nil tree root.
		node = &tree.Node{Kind: tree.KindFolder, Name: filepath.Base(root), RelPath: "."}
	}
	tr.Root = node
	return tr, nil
}

// buildFolder builds a folder node for dir, recursing into subdirectories.
// If the subtree rooted at dir contains no .md files at all, it returns
// nil (not an error) so the caller can prune that branch from the tree.
func buildFolder(dir, root string, mode Mode, tr *tree.Tree) (*tree.Node, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("walker: cannot read dir %q: %w", dir, err)
	}

	relDir, err := relSlash(root, dir)
	if err != nil {
		return nil, err
	}
	folder := &tree.Node{Kind: tree.KindFolder, Name: filepath.Base(dir), RelPath: relDir}
	// NOTE: a folder node is registered in tr.Index only when it absorbs a
	// README (ModeReadmeBody), because the index is keyed by the source path
	// of the Markdown file that backs a node. A folder with an empty
	// SourcePath (no README present, or mirror mode) has no backing file and
	// therefore is not indexed.

	hasMarkdown := false
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue // Skip dotfiles and dot-directories (e.g. .git, .DS_Store).
		}
		if e.Type()&os.ModeSymlink != 0 {
			continue // Do not follow symlinks; they can escape the tree or form cycles.
		}
		full := filepath.Join(dir, name)

		if e.IsDir() {
			child, err := buildFolder(full, root, mode, tr)
			if err != nil {
				return nil, err
			}
			if child != nil {
				folder.Children = append(folder.Children, child)
				hasMarkdown = true
			}
			continue
		}

		if !strings.HasSuffix(name, ".md") {
			continue
		}
		rel, err := relSlash(root, full)
		if err != nil {
			return nil, err
		}
		hasMarkdown = true

		if mode == ModeReadmeBody && name == readmeName {
			folder.SourcePath = rel
			tr.Index[rel] = folder
			continue
		}

		page := &tree.Node{
			Kind:       tree.KindPage,
			Name:       strings.TrimSuffix(name, ".md"),
			RelPath:    rel,
			SourcePath: rel,
		}
		folder.Children = append(folder.Children, page)
		tr.Index[rel] = page
	}

	if !hasMarkdown {
		return nil, nil
	}

	slices.SortStableFunc(folder.Children, func(a, b *tree.Node) int {
		return strings.Compare(a.Name, b.Name)
	})
	return folder, nil
}

// relSlash returns target as a path relative to root, using forward slashes
// regardless of the host OS path separator so that paths are stable and
// portable as index keys.
func relSlash(root, target string) (string, error) {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", fmt.Errorf("walker: cannot compute relative path for %q: %w", target, err)
	}
	return filepath.ToSlash(rel), nil
}
