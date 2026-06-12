package convert

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// nodeRenderer renders a goldmark AST into Confluence storage-format XHTML.
type nodeRenderer struct {
	resolver        LinkResolver
	mermaidMode     MermaidMode
	mermaidRenderer MermaidRenderer
	sink            AttachmentSink
}

// RegisterFuncs registers a render function for each kind of AST node.
func (r *nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// blocks
	reg.Register(ast.KindDocument, r.renderNoop)
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindTextBlock, r.renderNoop)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindCodeBlock, r.renderIndentedCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindListItem, r.renderListItem)
	reg.Register(ast.KindThematicBreak, r.renderThematicBreak)
	// inlines
	reg.Register(ast.KindText, r.renderText)
	reg.Register(ast.KindString, r.renderString)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindEmphasis, r.renderEmphasis)
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindAutoLink, r.renderAutoLink)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)
	// GFM table
	reg.Register(extast.KindTable, r.renderTable)
	reg.Register(extast.KindTableHeader, r.renderTableHeader)
	reg.Register(extast.KindTableRow, r.renderTableRow)
	reg.Register(extast.KindTableCell, r.renderTableCell)
}

func (r *nodeRenderer) renderNoop(util.BufWriter, []byte, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderHeading(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	level := strconv.Itoa(node.(*ast.Heading).Level)
	if entering {
		write(w, "<h"+level+">")
	} else {
		write(w, "</h"+level+">\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderParagraph(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		write(w, "<p>")
	} else {
		write(w, "</p>\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderBlockquote(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		write(w, "<blockquote>\n")
	} else {
		write(w, "</blockquote>\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderThematicBreak(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		write(w, "<hr/>\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.FencedCodeBlock)
	lang := ""
	if n.Info != nil {
		info := n.Info.Segment.Value(source)
		if f := bytes.Fields(info); len(f) > 0 {
			lang = string(f[0])
		}
	}
	code := codeBlockText(n, source)
	if lang == "mermaid" {
		if err := r.renderMermaid(w, code); err != nil {
			return ast.WalkStop, err
		}
		return ast.WalkSkipChildren, nil
	}
	writeCodeMacro(w, lang, code)
	return ast.WalkSkipChildren, nil
}

func (r *nodeRenderer) renderIndentedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	writeCodeMacro(w, "", codeBlockText(node, source))
	return ast.WalkSkipChildren, nil
}

func (r *nodeRenderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.HTMLBlock)
	if isTOCMarker(htmlBlockBytes(source, n)) {
		write(w, tocMacro())
		return ast.WalkSkipChildren, nil
	}
	writeLines(w, source, n.Lines())
	if n.HasClosure() {
		seg := n.ClosureLine
		writeBytes(w, seg.Value(source))
	}
	return ast.WalkSkipChildren, nil
}

func (r *nodeRenderer) renderList(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	tag := "ul"
	if node.(*ast.List).IsOrdered() {
		tag = "ol"
	}
	if entering {
		write(w, "<"+tag+">\n")
	} else {
		write(w, "</"+tag+">\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderListItem(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		write(w, "<li>")
	} else {
		write(w, "</li>\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Text)
	write(w, escapeXML(string(n.Segment.Value(source))))
	switch {
	case n.HardLineBreak():
		write(w, "<br/>\n")
	case n.SoftLineBreak():
		write(w, "\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderString(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		write(w, escapeXML(string(node.(*ast.String).Value)))
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	write(w, "<code>")
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			write(w, escapeXML(string(t.Segment.Value(source))))
		}
	}
	write(w, "</code>")
	return ast.WalkSkipChildren, nil
}

func (r *nodeRenderer) renderEmphasis(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	tag := "em"
	if node.(*ast.Emphasis).Level == 2 {
		tag = "strong"
	}
	if entering {
		write(w, "<"+tag+">")
	} else {
		write(w, "</"+tag+">")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderLink(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	res := r.resolve(string(node.(*ast.Link).Destination))
	if res.Internal {
		if entering {
			write(w, `<ac:link><ri:page ri:content-title="`+escapeXMLAttr(res.PageTitle)+`"/><ac:link-body>`)
		} else {
			write(w, `</ac:link-body></ac:link>`)
		}
		return ast.WalkContinue, nil
	}
	if entering {
		write(w, `<a href="`+escapeXMLAttr(res.Href)+`">`)
	} else {
		write(w, "</a>")
	}
	return ast.WalkContinue, nil
}

// resolve resolves an href through the configured resolver. When no resolver is
// set, it returns an external pass-through link (the href left unchanged).
func (r *nodeRenderer) resolve(href string) ResolvedLink {
	if r.resolver == nil {
		return ResolvedLink{Href: href}
	}
	return r.resolver.ResolveLink(href)
}

func (r *nodeRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.AutoLink)
	write(w, `<a href="`+escapeXMLAttr(string(n.URL(source)))+`">`+escapeXML(string(n.Label(source)))+"</a>")
	return ast.WalkSkipChildren, nil
}

func (r *nodeRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.Image)
	write(w, "<ac:image")
	if alt := textContent(n, source); alt != "" {
		write(w, ` ac:alt="`+escapeXMLAttr(alt)+`"`)
	}
	write(w, `><ri:url ri:value="`+escapeXMLAttr(string(n.Destination))+`"/></ac:image>`)
	return ast.WalkSkipChildren, nil
}

func (r *nodeRenderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.RawHTML)
	for i := 0; i < n.Segments.Len(); i++ {
		seg := n.Segments.At(i)
		writeBytes(w, seg.Value(source))
	}
	return ast.WalkSkipChildren, nil
}

func (r *nodeRenderer) renderTable(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		write(w, "<table><tbody>\n")
	} else {
		write(w, "</tbody></table>\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderTableHeader(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		write(w, "<tr>")
	} else {
		write(w, "</tr>\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderTableRow(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		write(w, "<tr>")
	} else {
		write(w, "</tr>\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderTableCell(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	tag := "td"
	if _, ok := node.Parent().(*extast.TableHeader); ok {
		tag = "th"
	}
	if entering {
		write(w, "<"+tag+">")
	} else {
		write(w, "</"+tag+">")
	}
	return ast.WalkContinue, nil
}

// --- helpers ---

// write writes s to the BufWriter, ignoring the write error. The error never
// occurs in practice because the writer is an in-memory buffer.
func write(w util.BufWriter, s string) {
	_, _ = w.WriteString(s)
}

// writeBytes is the []byte version of write.
func writeBytes(w util.BufWriter, b []byte) {
	_, _ = w.Write(b)
}

func codeBlockText(n ast.Node, source []byte) []byte {
	var b bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(source))
	}
	return b.Bytes()
}

func writeLines(w util.BufWriter, source []byte, lines *text.Segments) {
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		writeBytes(w, seg.Value(source))
	}
}

// htmlBlockBytes returns the raw bytes of an HTML block — its content lines plus
// the closing line, if any. Used to detect special markers such as the TOC comment.
func htmlBlockBytes(source []byte, n *ast.HTMLBlock) []byte {
	var b bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(source))
	}
	if n.HasClosure() {
		seg := n.ClosureLine
		b.Write(seg.Value(source))
	}
	return b.Bytes()
}

func writeCodeMacro(w util.BufWriter, lang string, body []byte) {
	write(w, `<ac:structured-macro ac:name="code">`)
	if lang != "" {
		write(w, `<ac:parameter ac:name="language">`+escapeXML(lang)+`</ac:parameter>`)
	}
	write(w, `<ac:plain-text-body>`)
	write(w, cdata(body))
	write(w, "</ac:plain-text-body></ac:structured-macro>\n")
}

func cdata(body []byte) string {
	s := strings.ReplaceAll(string(body), "]]>", "]]]]><![CDATA[>")
	return "<![CDATA[" + s + "]]>"
}

func textContent(n ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch t := c.(type) {
			case *ast.Text:
				b.Write(t.Segment.Value(source))
			case *ast.String:
				b.Write(t.Value)
			}
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func escapeXML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

func escapeXMLAttr(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}
