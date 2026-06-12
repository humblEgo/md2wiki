package convert

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

// runGolden converts testdata/<name>.md and compares the result against
// testdata/<name>.golden. When the -update flag is set, it regenerates the
// golden file instead of comparing.
func runGolden(t *testing.T, name string) {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", name+".md"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Document(src)
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	goldenPath := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run with -update first): %v", err)
	}
	if got != string(want) {
		t.Errorf("%s mismatch:\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
	}
}

func TestGoldenBasics(t *testing.T) { runGolden(t, "basics") }

func TestGoldenCode(t *testing.T) { runGolden(t, "code") }

func TestGoldenLists(t *testing.T) { runGolden(t, "lists") }

func TestGoldenTable(t *testing.T) { runGolden(t, "table") }

func TestGoldenLinksImages(t *testing.T) { runGolden(t, "links_images") }

func TestGoldenComposite(t *testing.T) { runGolden(t, "composite") }

func TestGoldenTOC(t *testing.T) { runGolden(t, "toc") }

func TestEscape_TextAndAttr(t *testing.T) {
	got, err := Document([]byte("# A < B & C\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "<h1>A &lt; B &amp; C</h1>") {
		t.Errorf("heading text must be XML-escaped, got %q", got)
	}
}

func TestEscape_LinkHref(t *testing.T) {
	got, err := Document([]byte("[x](https://e.com?a=1&b=2)\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `href="https://e.com?a=1&amp;b=2"`) {
		t.Errorf("href must be attribute-escaped, got %q", got)
	}
}

func TestRawHTML_Passthrough(t *testing.T) {
	got, err := Document([]byte("<details><summary>x</summary>y</details>\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "<details><summary>x</summary>y</details>") {
		t.Errorf("raw HTML must pass through verbatim, got %q", got)
	}
}

func TestCodeMacro_Language(t *testing.T) {
	got, err := Document([]byte("```go\nx := 1\n```\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := `<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">go</ac:parameter><ac:plain-text-body><![CDATA[x := 1
]]></ac:plain-text-body></ac:structured-macro>` + "\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCodeMacro_NoLanguage(t *testing.T) {
	got, err := Document([]byte("```\nplain\n```\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, `ac:name="language"`) {
		t.Errorf("no-language code block must not emit a language parameter, got %q", got)
	}
}

func TestCodeMacro_CDATASplit(t *testing.T) {
	got, err := Document([]byte("```\na]]>b\n```\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `<![CDATA[a]]]]><![CDATA[>b`) {
		t.Errorf("CDATA terminator inside body must be split, got %q", got)
	}
}

type stubResolver struct{ m map[string]ResolvedLink }

func (s stubResolver) ResolveLink(href string) ResolvedLink {
	if r, ok := s.m[href]; ok {
		return r
	}
	return ResolvedLink{Href: href}
}

func TestDocument_InternalLink(t *testing.T) {
	r := stubResolver{m: map[string]ResolvedLink{
		"other.md": {Internal: true, PageTitle: "Other Page"},
	}}
	got, err := Document([]byte("[label](other.md)\n"), WithLinkResolver(r))
	if err != nil {
		t.Fatal(err)
	}
	want := `<p><ac:link><ri:page ri:content-title="Other Page"/><ac:link-body>label</ac:link-body></ac:link></p>` + "\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDocument_ExternalLinkWithResolver(t *testing.T) {
	got, err := Document([]byte("[x](https://e.com)\n"), WithLinkResolver(stubResolver{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `<a href="https://e.com">x</a>`) {
		t.Errorf("external link should pass through, got %q", got)
	}
}

func TestDocument_NoResolver_AllExternal(t *testing.T) {
	got, err := Document([]byte("[x](other.md)\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `<a href="other.md">x</a>`) {
		t.Errorf("without resolver all links are external, got %q", got)
	}
}

func TestDocument_InternalLink_TitleEscaped(t *testing.T) {
	r := stubResolver{m: map[string]ResolvedLink{
		"p.md": {Internal: true, PageTitle: `A & "B"`},
	}}
	got, err := Document([]byte("[x](p.md)\n"), WithLinkResolver(r))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `ri:content-title="A &amp; &quot;B&quot;"`) {
		t.Errorf("page title must be attribute-escaped, got %q", got)
	}
}

type fakeRenderer struct {
	png   []byte
	calls int
}

func (f *fakeRenderer) Render(source []byte) ([]byte, error) {
	f.calls++
	return f.png, nil
}

// compile-time interface check; fakeRenderer is shared across the mermaid tests below.
var _ MermaidRenderer = (*fakeRenderer)(nil)

func TestMermaid_Raw(t *testing.T) {
	out, err := Document([]byte("```mermaid\ngraph TD\nA-->B\n```\n"), WithMermaidMode(MermaidRaw))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">mermaid</ac:parameter><ac:plain-text-body><![CDATA[graph TD`) {
		t.Errorf("raw mode should emit a mermaid code macro, got %q", out)
	}
	if strings.Contains(out, "ri:attachment") {
		t.Errorf("raw mode must not emit an attachment image, got %q", out)
	}
}

func TestMermaid_Render(t *testing.T) {
	fr := &fakeRenderer{png: []byte("PNGDATA")}
	var got []Attachment
	out, err := Document([]byte("```mermaid\ngraph TD;A-->B\n```\n"),
		WithMermaidMode(MermaidRender),
		WithMermaidRenderer(fr),
		WithAttachmentSink(func(a Attachment) { got = append(got, a) }),
	)
	if err != nil {
		t.Fatal(err)
	}
	if fr.calls != 1 {
		t.Errorf("renderer calls = %d, want 1", fr.calls)
	}
	if len(got) != 1 {
		t.Fatalf("attachments = %d, want 1", len(got))
	}
	if !strings.HasPrefix(got[0].Filename, "mermaid-") || !strings.HasSuffix(got[0].Filename, ".png") {
		t.Errorf("filename = %q, want mermaid-<hash>.png", got[0].Filename)
	}
	if string(got[0].Data) != "PNGDATA" {
		t.Errorf("attachment data = %q, want PNGDATA", got[0].Data)
	}
	if !strings.Contains(out, `<ac:image><ri:attachment ri:filename="`+got[0].Filename+`"/></ac:image>`) {
		t.Errorf("missing attachment image tag, got %q", out)
	}
	if strings.Contains(out, "expand") {
		t.Errorf("render mode must not emit an expand macro, got %q", out)
	}
}

func TestMermaid_Details(t *testing.T) {
	fr := &fakeRenderer{png: []byte("X")}
	var got []Attachment
	out, err := Document([]byte("```mermaid\ngraph TD;A-->B\n```\n"),
		WithMermaidRenderer(fr), // no mode set, so the default (details) applies
		WithAttachmentSink(func(a Attachment) { got = append(got, a) }),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("attachments = %d, want 1", len(got))
	}
	if !strings.Contains(out, "ri:attachment") {
		t.Errorf("details must emit the image, got %q", out)
	}
	if !strings.Contains(out, `<ac:structured-macro ac:name="expand"><ac:parameter ac:name="title">Mermaid source</ac:parameter><ac:rich-text-body>`) {
		t.Errorf("details must emit an expand macro, got %q", out)
	}
	if !strings.Contains(out, `<ac:parameter ac:name="language">mermaid</ac:parameter>`) {
		t.Errorf("details expand must contain the mermaid code macro, got %q", out)
	}
}

func TestMermaid_RenderWithoutRenderer_Errors(t *testing.T) {
	_, err := Document([]byte("```mermaid\ngraph TD;A-->B\n```\n"), WithMermaidMode(MermaidRender))
	if err == nil {
		t.Fatal("render mode without a renderer must return an error")
	}
}

func TestMermaid_FilenameDeterministic(t *testing.T) {
	fr := &fakeRenderer{png: []byte("X")}
	in := []byte("```mermaid\ngraph TD;A-->B\n```\n")
	collect := func() string {
		var got []Attachment
		_, err := Document(in, WithMermaidMode(MermaidRender), WithMermaidRenderer(fr),
			WithAttachmentSink(func(a Attachment) { got = append(got, a) }))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("attachments = %d, want 1", len(got))
		}
		return got[0].Filename
	}
	if a, b := collect(), collect(); a != b {
		t.Errorf("filename not deterministic: %q vs %q", a, b)
	}
}
