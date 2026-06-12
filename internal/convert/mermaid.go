package convert

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/yuin/goldmark/util"
)

// renderMermaid emits the Mermaid source into the storage format according to
// the configured mode (raw, render, or details).
func (r *nodeRenderer) renderMermaid(w util.BufWriter, code []byte) error {
	if r.mermaidMode == MermaidRaw {
		writeCodeMacro(w, "mermaid", code)
		return nil
	}
	if r.mermaidRenderer == nil {
		return fmt.Errorf("convert: mermaid %v 모드는 렌더러가 필요하다(WithMermaidRenderer)", r.mermaidMode)
	}
	png, err := r.mermaidRenderer.Render(code)
	if err != nil {
		return fmt.Errorf("convert: mermaid 렌더 실패: %w", err)
	}
	name := mermaidFilename(code)
	if r.sink != nil {
		r.sink(Attachment{Filename: name, Data: png})
	}
	write(w, `<ac:image><ri:attachment ri:filename="`+escapeXMLAttr(name)+`"/></ac:image>`+"\n")
	if r.mermaidMode == MermaidDetails {
		write(w, `<ac:structured-macro ac:name="expand"><ac:parameter ac:name="title">Mermaid source</ac:parameter><ac:rich-text-body>`)
		writeCodeMacro(w, "mermaid", code)
		write(w, "</ac:rich-text-body></ac:structured-macro>\n")
	}
	return nil
}

// mermaidFilename builds a deterministic PNG filename from a hash of the source
// content, so that identical diagrams produce the same filename across runs
// (idempotent attachment naming).
func mermaidFilename(code []byte) string {
	sum := sha256.Sum256(code)
	return "mermaid-" + hex.EncodeToString(sum[:])[:12] + ".png"
}
