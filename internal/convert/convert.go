// Package convert turns Markdown into the Confluence storage format (an XHTML
// fragment), including link rewriting and Mermaid diagram handling.
package convert

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var mdParser = goldmark.New(goldmark.WithExtensions(extension.Table))

// ResolvedLink is the result of resolving a single link.
type ResolvedLink struct {
	Internal  bool   // true means an internal link to another Confluence page
	PageTitle string // target page title, set only when Internal is true
	Href      string // href to use for an external link (usually the original, unchanged)
}

// LinkResolver resolves an original href into a ResolvedLink, deciding whether
// it points to another mirrored Confluence page or stays an external link.
type LinkResolver interface {
	ResolveLink(href string) ResolvedLink
}

type options struct {
	resolver        LinkResolver
	mermaidMode     MermaidMode
	mermaidRenderer MermaidRenderer
	sink            AttachmentSink
}

// Option configures the behavior of Document.
type Option func(*options)

// WithLinkResolver sets the link resolver used during conversion.
func WithLinkResolver(r LinkResolver) Option {
	return func(o *options) { o.resolver = r }
}

// MermaidMode decides how ```mermaid blocks are handled.
type MermaidMode int

const (
	// MermaidDetails emits the rendered image plus the raw source in a
	// collapsible section (the default).
	MermaidDetails MermaidMode = iota
	// MermaidRender emits only the rendered image.
	MermaidRender
	// MermaidRaw emits only the raw source as a code block, with no rendering.
	MermaidRaw
)

// Attachment is a binary to attach to the page, such as a rendered diagram.
type Attachment struct {
	Filename string
	Data     []byte
}

// AttachmentSink collects attachments produced during conversion. Conversion
// only emits them here; the actual upload to Confluence is handled by sync.
type AttachmentSink func(Attachment)

// MermaidRenderer renders Mermaid source into PNG bytes.
type MermaidRenderer interface {
	Render(source []byte) (png []byte, err error)
}

// WithMermaidMode sets the mode used to handle ```mermaid blocks (defaults to MermaidDetails).
func WithMermaidMode(m MermaidMode) Option {
	return func(o *options) { o.mermaidMode = m }
}

// WithMermaidRenderer sets the renderer used in the render and details modes.
func WithMermaidRenderer(r MermaidRenderer) Option {
	return func(o *options) { o.mermaidRenderer = r }
}

// WithAttachmentSink sets the sink that receives rendered attachments.
func WithAttachmentSink(s AttachmentSink) Option {
	return func(o *options) { o.sink = s }
}

// Document converts a Markdown body into a Confluence storage-format XHTML body
// fragment. When no resolver is supplied, every link is treated as external and
// emitted unchanged.
func Document(src []byte, opts ...Option) (string, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	doc := mdParser.Parser().Parse(text.NewReader(src))
	rend := renderer.NewRenderer(renderer.WithNodeRenderers(
		util.Prioritized(&nodeRenderer{
			resolver:        o.resolver,
			mermaidMode:     o.mermaidMode,
			mermaidRenderer: o.mermaidRenderer,
			sink:            o.sink,
		}, 1000),
	))
	var buf bytes.Buffer
	if err := rend.Render(&buf, src, doc); err != nil {
		return "", fmt.Errorf("convert: render failed: %w", err)
	}
	return buf.String(), nil
}
