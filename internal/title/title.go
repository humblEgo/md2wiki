// Package title derives the page title for each document-tree node and resolves
// collisions between nodes that would otherwise end up with the same title.
package title

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/humblEgo/md2wiki/internal/tree"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// extractH1 returns the trimmed text of the first level-1 heading found in the
// Markdown source. It returns ok=false when there is no level-1 heading or when
// that heading's text is empty after trimming.
func extractH1(src []byte) (string, bool) {
	doc := goldmark.New().Parser().Parse(text.NewReader(src))

	var heading *ast.Heading
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || heading != nil {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok && h.Level == 1 {
			heading = h
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	if heading == nil {
		return "", false
	}

	title := strings.TrimSpace(headingText(heading, src))
	if title == "" {
		return "", false
	}
	return title, true
}

// headingText concatenates only the inline text of a heading node, leaving out
// the Markdown markers themselves (for example "#", "*", or "`").
func headingText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(src))
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// Resolve assigns a unique Title to every node in the tree. When two or more
// nodes share the same base title, every node in that colliding group gets a
// " (<RelPath>)" suffix appended. Because the suffix is the node's own relative
// path rather than, say, a running counter, each title is deterministic and
// stable: a given node keeps the same title regardless of how many other nodes
// happen to collide with it or what order they are visited in.
func Resolve(root string, t *tree.Tree) error {
	if t == nil || t.Root == nil {
		return nil
	}
	bases := make(map[*tree.Node]string)
	counts := make(map[string]int)
	if err := collectBases(root, t.Root, bases, counts); err != nil {
		return err
	}
	seen := make(map[string]struct{})
	assignTitles(t.Root, bases, counts, seen)
	return nil
}

// collectBases walks the tree depth-first, computing each node's base title and
// recording it in bases, while counts tallies how many nodes share each base
// title. The counts let assignTitles tell, in a second pass, which base titles
// collide and therefore need the disambiguating suffix.
func collectBases(root string, n *tree.Node, bases map[*tree.Node]string, counts map[string]int) error {
	b, err := baseTitle(root, n)
	if err != nil {
		return err
	}
	bases[n] = b
	counts[b]++
	for _, c := range n.Children {
		if err := collectBases(root, c, bases, counts); err != nil {
			return err
		}
	}
	return nil
}

// assignTitles walks the tree depth-first and sets each node's Title. When a
// node's base title collides with another node's (its count is greater than 1),
// the node's " (<RelPath>)" suffix is appended to disambiguate it. The result is
// then passed through uniqueTitle as a final safety net against any remaining
// duplicates.
func assignTitles(n *tree.Node, bases map[*tree.Node]string, counts map[string]int, seen map[string]struct{}) {
	desired := bases[n]
	if counts[desired] > 1 {
		desired += " (" + n.RelPath + ")"
	}
	n.Title = uniqueTitle(desired, seen)
	for _, c := range n.Children {
		assignTitles(c, bases, counts, seen)
	}
}

// baseTitle derives a node's base title (its title before any disambiguation).
// If the node has a SourcePath, the title is the first H1 heading of that .md
// file; if the node has no SourcePath, or its source has no usable H1, the title
// falls back to the node's Name.
func baseTitle(root string, n *tree.Node) (string, error) {
	if n.SourcePath == "" {
		return n.Name, nil
	}
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(n.SourcePath)))
	if err != nil {
		return "", fmt.Errorf("title: cannot read %q: %w", n.SourcePath, err)
	}
	if h1, ok := extractH1(src); ok {
		return h1, nil
	}
	return n.Name, nil
}

// uniqueTitle returns desired unchanged if it has not been seen yet; otherwise
// it appends " (2)", " (3)", and so on until it finds an unused title. This is a
// safety net for the rare case where the path-suffixed titles produced by
// assignTitles still collide literally (for example, distinct nodes whose names
// already contain parenthesized paths). Every returned title is recorded in seen.
func uniqueTitle(desired string, seen map[string]struct{}) string {
	if _, ok := seen[desired]; !ok {
		seen[desired] = struct{}{}
		return desired
	}
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s (%d)", desired, i)
		if _, ok := seen[cand]; !ok {
			seen[cand] = struct{}{}
			return cand
		}
	}
}
