package wizard

import "testing"

// fakePrompter pops canned answers per method type, in call order within each type.
type fakePrompter struct {
	inputs    []string
	passwords []string
	selects   []string
	confirms  []bool
}

func (f *fakePrompter) Input(_, _ string, validate func(string) error) (string, error) {
	v := f.inputs[0]
	f.inputs = f.inputs[1:]
	if validate != nil {
		if err := validate(v); err != nil {
			return "", err
		}
	}
	return v, nil
}

func (f *fakePrompter) Password(string) (string, error) {
	v := f.passwords[0]
	f.passwords = f.passwords[1:]
	return v, nil
}

func (f *fakePrompter) Select(string, []string) (string, error) {
	v := f.selects[0]
	f.selects = f.selects[1:]
	return v, nil
}

func (f *fakePrompter) Confirm(string, bool) (bool, error) {
	v := f.confirms[0]
	f.confirms = f.confirms[1:]
	return v, nil
}

func TestRun_SingleMapping(t *testing.T) {
	p := &fakePrompter{
		// Input: source, space, rootPage, baseURL, email
		inputs:    []string{"docs", "DOCS", "", "https://x.atlassian.net", "a@b.com"},
		passwords: []string{"tok-123"},
		// Select: layout, mermaid
		selects: []string{"readme-body", "details"},
		// Confirm: banner, add-more, open-browser
		confirms: []bool{true, false, true},
	}
	var opened string
	res, err := Run(p, func(u string) error { opened = u; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if opened != TokenPageURL {
		t.Errorf("opened = %q, want %q", opened, TokenPageURL)
	}
	if res.Token != "tok-123" {
		t.Errorf("token = %q", res.Token)
	}
	if res.File.BaseURL != "https://x.atlassian.net" || res.File.Email != "a@b.com" {
		t.Errorf("conn = %+v", res.File)
	}
	if res.File.LayoutMode != "readme-body" || res.File.MermaidMode != "details" {
		t.Errorf("defaults = %+v", res.File)
	}
	if res.File.Banner == nil || *res.File.Banner != true {
		t.Errorf("banner = %v", res.File.Banner)
	}
	if len(res.File.Mappings) != 1 {
		t.Fatalf("mappings = %d, want 1", len(res.File.Mappings))
	}
	if res.File.Mappings[0].Source != "docs" || res.File.Mappings[0].Space != "DOCS" {
		t.Errorf("m0 = %+v", res.File.Mappings[0])
	}
}

func TestRun_MultipleMappings_SkipBrowser(t *testing.T) {
	p := &fakePrompter{
		// Input: m1(source,space,root), m2(source,space,root), baseURL, email
		inputs:    []string{"docs", "DOCS", "111", "ops", "OPS", "", "https://x.atlassian.net", "a@b.com"},
		passwords: []string{""},
		selects:   []string{"mirror", "raw"},
		// Confirm: banner, add-more(yes), add-more(no), open-browser(no)
		confirms: []bool{false, true, false, false},
	}
	var opened string
	res, err := Run(p, func(u string) error { opened = u; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if opened != "" {
		t.Errorf("browser should not open when declined, got %q", opened)
	}
	if res.Token != "" {
		t.Errorf("token should be empty, got %q", res.Token)
	}
	if res.File.Banner == nil || *res.File.Banner != false {
		t.Errorf("banner = %v, want *false", res.File.Banner)
	}
	if len(res.File.Mappings) != 2 {
		t.Fatalf("mappings = %d, want 2", len(res.File.Mappings))
	}
	if res.File.Mappings[0].RootPage != "111" || res.File.Mappings[1].Space != "OPS" {
		t.Errorf("mappings = %+v", res.File.Mappings)
	}
}
