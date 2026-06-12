package convert

import (
	"strings"
	"testing"
)

func TestIsTOCMarker(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"<!-- toc -->", true},
		{"<!-- TOC -->", true},
		{"<!--  toc  -->", true},
		{"<!-- toc -->\n", true},
		{"<!--toc-->", true},
		{"<!-- toc: x -->", false},
		{"<!-- toc extra -->", false},
		{"<!-- hello -->", false},
		{"<!-->", false},
		{"not a comment", false},
		{"<p>toc</p>", false},
	}
	for _, c := range cases {
		if got := isTOCMarker([]byte(c.in)); got != c.want {
			t.Errorf("isTOCMarker(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestTocMacro(t *testing.T) {
	if got, want := tocMacro(), `<ac:structured-macro ac:name="toc"/>`+"\n"; got != want {
		t.Errorf("tocMacro() = %q, want %q", got, want)
	}
}

const tocMacroStr = `<ac:structured-macro ac:name="toc"/>`

func TestTOC_MarkerReplaced(t *testing.T) {
	got, err := Document([]byte("<!-- toc -->\n\n# Title\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, tocMacroStr) {
		t.Errorf("marker should become the toc macro, got %q", got)
	}
	if strings.Contains(got, "<!-- toc -->") {
		t.Errorf("raw marker comment must not remain, got %q", got)
	}
}

func TestTOC_MarkerVariants(t *testing.T) {
	for _, m := range []string{"<!-- toc -->", "<!-- TOC -->", "<!--  toc  -->"} {
		got, err := Document([]byte(m + "\n\n# H\n"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, tocMacroStr) {
			t.Errorf("variant %q should produce the toc macro, got %q", m, got)
		}
	}
}

func TestTOC_NonMarkerCommentPassThrough(t *testing.T) {
	for _, c := range []string{"<!-- keep me -->", "<!-- toc: x -->"} {
		got, err := Document([]byte(c + "\n"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, c) {
			t.Errorf("non-marker comment %q must pass through, got %q", c, got)
		}
		if strings.Contains(got, tocMacroStr) {
			t.Errorf("non-marker comment %q must not produce the toc macro, got %q", c, got)
		}
	}
}

func TestTOC_InPlacePosition(t *testing.T) {
	got, err := Document([]byte("# A\n\n<!-- toc -->\n\n## B\n"))
	if err != nil {
		t.Fatal(err)
	}
	macro := strings.Index(got, tocMacroStr)
	h1 := strings.Index(got, "<h1>A</h1>")
	h2 := strings.Index(got, "<h2>B</h2>")
	if h1 < 0 || macro <= h1 || h2 <= macro {
		t.Errorf("macro should sit between H1 and H2; h1=%d macro=%d h2=%d, out=%q", h1, macro, h2, got)
	}
}

func TestTOC_MultipleMarkers(t *testing.T) {
	got, err := Document([]byte("<!-- toc -->\n\ntext\n\n<!-- toc -->\n"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(got, tocMacroStr); n != 2 {
		t.Errorf("expected 2 toc macros, got %d in %q", n, got)
	}
}
