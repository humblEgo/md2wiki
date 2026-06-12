package convert

import "strings"

// tocMacro returns the Confluence storage-format snippet for the native table-
// of-contents macro. Confluence builds the list from the page's headings.
func tocMacro() string {
	return `<ac:structured-macro ac:name="toc"/>` + "\n"
}

// isTOCMarker reports whether an HTML block is the TOC marker comment. It
// matches an HTML comment whose sole content is the token "toc"
// (case-insensitive, surrounding whitespace ignored): <!-- toc -->, <!-- TOC -->,
// <!--  toc  -->. Any other comment or content is not a marker.
func isTOCMarker(block []byte) bool {
	s := strings.TrimSpace(string(block))
	// 7 == len("<!--") + len("-->"); the guard also keeps the slice below safe.
	if len(s) < 7 || !strings.HasPrefix(s, "<!--") || !strings.HasSuffix(s, "-->") {
		return false
	}
	inner := strings.TrimSpace(s[len("<!--") : len(s)-len("-->")])
	return strings.EqualFold(inner, "toc")
}
