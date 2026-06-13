package wizard

import "testing"

func TestParseDestination(t *testing.T) {
	tests := []struct {
		in         string
		wantKey    string
		wantParent string
	}{
		{"DOCS", "DOCS", ""},
		{"  DOCS  ", "DOCS", ""},
		{"https://x.atlassian.net/wiki/spaces/DOCS", "DOCS", ""},
		{"https://x.atlassian.net/wiki/spaces/DOCS/overview", "DOCS", ""},
		{"https://x.atlassian.net/wiki/spaces/DOCS/pages/123456/Home", "DOCS", "123456"},
		{"  https://x.atlassian.net/wiki/spaces/DOCS/pages/123/T  ", "DOCS", "123"},
		{"https://x.atlassian.net/wiki/spaces/DOCS?src=sidebar", "DOCS", ""},
		{"https://x.atlassian.net/wiki/spaces/~7120abc/pages/9/P", "~7120abc", "9"},
		{"", "", ""},
	}
	for _, tt := range tests {
		key, parent := parseDestination(tt.in)
		if key != tt.wantKey || parent != tt.wantParent {
			t.Errorf("parseDestination(%q) = (%q, %q), want (%q, %q)", tt.in, key, parent, tt.wantKey, tt.wantParent)
		}
	}
}

func TestValidateDestination(t *testing.T) {
	ok := []string{
		"DOCS",
		"~7120abc",
		"https://x.atlassian.net/wiki/spaces/OPS",
		"https://x.atlassian.net/wiki/spaces/DOCS/pages/9/Home",
	}
	for _, s := range ok {
		if err := validateDestination(s); err != nil {
			t.Errorf("validateDestination(%q) = %v, want nil", s, err)
		}
	}
	bad := []string{
		"",
		"   ",
		"https://x.atlassian.net/wiki/spaces", // no key after /spaces
		"https://x.atlassian.net/wiki",        // not a space URL at all
	}
	for _, s := range bad {
		if err := validateDestination(s); err == nil {
			t.Errorf("validateDestination(%q) = nil, want error", s)
		}
	}
}
