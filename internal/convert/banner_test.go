package convert

import (
	"strings"
	"testing"
)

func TestMirrorBanner(t *testing.T) {
	b := MirrorBanner()
	for _, want := range []string{
		`<ac:structured-macro ac:name="info">`,
		`<ac:rich-text-body>`,
		`<strong>Do not edit it here</strong>`,
		`mirrored from a Git repository`,
		`overwritten on the next sync`,
		`</ac:rich-text-body></ac:structured-macro>`,
	} {
		if !strings.Contains(b, want) {
			t.Errorf("banner missing %q\ngot: %s", want, b)
		}
	}
}
