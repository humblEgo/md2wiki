package convert

import "testing"

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
