// Package link resolves relative .md links into Confluence page links using the
// document tree's index.
package link

import (
	"path"
	"regexp"
	"strings"

	"github.com/humblEgo/md2wiki/internal/convert"
	"github.com/humblEgo/md2wiki/internal/tree"
)

var schemeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

type resolver struct {
	index   map[string]*tree.Node
	baseDir string
}

// NewResolver builds a link resolver from the tree index and the current
// document's path (a slash-separated path relative to the tree root). The
// document's directory becomes the base against which relative links resolve.
func NewResolver(t *tree.Tree, currentDoc string) convert.LinkResolver {
	return &resolver{index: t.Index, baseDir: path.Dir(currentDoc)}
}

// ResolveLink resolves an href into either an internal page link or an external
// link. Empty hrefs, protocol-relative ("//") links, pure fragments ("#..."),
// links carrying a URL scheme, and any target that is not a ".md" file or is not
// found in the tree index are treated as external and passed through unchanged.
func (r *resolver) ResolveLink(href string) convert.ResolvedLink {
	external := convert.ResolvedLink{Href: href}
	if href == "" || strings.HasPrefix(href, "//") || strings.HasPrefix(href, "#") || schemeRe.MatchString(href) {
		return external
	}
	base := href
	if i := strings.IndexByte(base, '#'); i >= 0 {
		base = base[:i]
	}
	if !strings.HasSuffix(base, ".md") {
		return external
	}
	target := path.Clean(path.Join(r.baseDir, base))
	node, ok := r.index[target]
	if !ok {
		return external
	}
	return convert.ResolvedLink{Internal: true, PageTitle: node.Title}
}
