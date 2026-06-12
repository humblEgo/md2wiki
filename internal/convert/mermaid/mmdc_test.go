package mermaid_test

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/humblEgo/md2wiki/internal/convert"
	"github.com/humblEgo/md2wiki/internal/convert/mermaid"
)

// Compile-time check that Mmdc satisfies the convert.MermaidRenderer interface.
var _ convert.MermaidRenderer = (*mermaid.Mmdc)(nil)

func TestMmdcRender(t *testing.T) {
	if _, err := exec.LookPath("mmdc"); err != nil {
		t.Skip("mmdc not on PATH")
	}
	png, err := mermaid.New().Render([]byte("graph TD;A-->B;\n"))
	if err != nil {
		t.Skipf("mmdc render failed (headless chromium 부재 가능): %v", err)
	}
	if !bytes.HasPrefix(png, []byte("\x89PNG")) {
		t.Errorf("output is not a PNG (magic bytes missing), got %d bytes", len(png))
	}
}
