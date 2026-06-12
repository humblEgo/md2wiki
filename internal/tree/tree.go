// Package tree defines the data types for the document tree that the walker builds.
package tree

// Kind distinguishes whether a node is a folder page or a single-document page.
type Kind int

const (
	// KindFolder is a folder page that corresponds to a directory.
	KindFolder Kind = iota
	// KindPage is a single .md document page.
	KindPage
)

// Node represents a single page to be mirrored to Confluence.
type Node struct {
	Kind Kind
	// Name is the directory name for a folder, or the filename with the ".md"
	// suffix removed for a page.
	Name string
	// RelPath is the slash-separated path relative to the root. It is the
	// directory path for a folder, the .md path for a page, and "." for the root.
	RelPath string
	// SourcePath is the slash-separated, root-relative path of the .md file that
	// becomes this node's body. For a folder in ModeReadmeBody it is the README.md
	// inside that directory (or "" if there is none); for a folder in ModeMirror it
	// is always ""; for a page it is that page's .md file path.
	SourcePath string
	// Title is the final page title assigned by title.Resolve. It is "" before
	// Resolve runs.
	Title string
	// Children are the children of a folder node, sorted by Name in byte order.
	Children []*Node
}

// Tree is the whole document tree that the walker builds.
type Tree struct {
	// Root is the top-level folder node corresponding to the root directory.
	Root *Node
	// Index maps a .md relative path to its *Node. It is used by later passes
	// (title, link, and pageId resolution) for O(1) lookups.
	Index map[string]*Node
}
