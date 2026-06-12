// Package mermaid renders Mermaid diagrams into PNGs by shelling out to the
// external mermaid CLI (mmdc).
package mermaid

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Mmdc renders Mermaid source into PNGs by invoking the mmdc binary. It
// structurally satisfies the convert.MermaidRenderer interface.
type Mmdc struct {
	Bin string // path to the executable (defaults to "mmdc")
}

// New creates an Mmdc with the default configuration.
func New() *Mmdc {
	return &Mmdc{Bin: "mmdc"}
}

// Render renders Mermaid source into PNG bytes.
func (m *Mmdc) Render(source []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "md2wiki-mermaid-")
	if err != nil {
		return nil, fmt.Errorf("mermaid: 임시 디렉토리: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	in := filepath.Join(dir, "input.mmd")
	out := filepath.Join(dir, "output.png")
	if err := os.WriteFile(in, source, 0o644); err != nil {
		return nil, fmt.Errorf("mermaid: 입력 쓰기: %w", err)
	}

	cmd := exec.Command(m.Bin, "-i", in, "-o", out)
	if combined, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("mermaid: mmdc 실패: %w: %s", err, combined)
	}

	png, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("mermaid: 출력 읽기: %w", err)
	}
	return png, nil
}
