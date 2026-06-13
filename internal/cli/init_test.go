package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/humblEgo/md2wiki/internal/wizard"
)

// stubPrompter returns canned answers; used to drive runInit without a TTY.
type stubPrompter struct {
	inputs    []string
	passwords []string
	selects   []string
	confirms  []bool
}

func (s *stubPrompter) Input(_, _ string, _ func(string) error) (string, error) {
	v := s.inputs[0]
	s.inputs = s.inputs[1:]
	return v, nil
}
func (s *stubPrompter) Password(string) (string, error) {
	v := s.passwords[0]
	s.passwords = s.passwords[1:]
	return v, nil
}
func (s *stubPrompter) Select(string, []wizard.Choice) (string, error) {
	v := s.selects[0]
	s.selects = s.selects[1:]
	return v, nil
}
func (s *stubPrompter) Confirm(string, string, bool) (bool, error) {
	v := s.confirms[0]
	s.confirms = s.confirms[1:]
	return v, nil
}

func baseDeps(out *bytes.Buffer) initDeps {
	return initDeps{
		prompter:    &stubPrompter{},
		openBrowser: func(string) error { return nil },
		verify:      func(context.Context, string, string, string, string) error { return nil },
		out:         out,
		fileExists:  func(string) bool { return false },
		writeFile:   func(string, []byte) error { return nil },
	}
}

func TestRunInit_WritesFileAndVerifies(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "md2wiki.yaml")
	var out bytes.Buffer
	verified := ""
	d := baseDeps(&out)
	d.prompter = &stubPrompter{
		// Input: source, destination, baseURL, email
		inputs:    []string{"docs", "DOCS", "https://x.atlassian.net", "a@b.com"},
		passwords: []string{"tok-123"},
		selects:   []string{"readme-body", "details"},
		// Confirm: banner, add-more, open-browser
		confirms: []bool{true, false, true},
	}
	d.verify = func(_ context.Context, _, _, _, space string) error { verified = space; return nil }
	d.fileExists = func(p string) bool { _, err := os.Stat(p); return err == nil }
	d.writeFile = func(p string, data []byte) error { return os.WriteFile(target, data, 0o600) }

	if err := runInit(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "space: DOCS") {
		t.Errorf("written file missing mapping:\n%s", data)
	}
	if strings.Contains(string(data), "tok-123") {
		t.Errorf("token must never be written to file:\n%s", data)
	}
	if verified != "DOCS" {
		t.Errorf("verify space = %q, want DOCS", verified)
	}
	if !strings.Contains(out.String(), "export MD2WIKI_API_TOKEN='tok-123'") {
		t.Errorf("output missing export line:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "md2wiki apply") {
		t.Errorf("output missing apply hint:\n%s", out.String())
	}
}

func TestRunInit_NewPathWhenExists(t *testing.T) {
	var out bytes.Buffer
	written := map[string][]byte{}
	d := baseDeps(&out)
	d.prompter = &stubPrompter{
		inputs: []string{
			"docs", "DOCS", "https://x.atlassian.net", "a@b.com", // wizard
			"custom.yaml", // overwrite=No → new path
		},
		passwords: []string{""},
		selects:   []string{"readme-body", "details"},
		// banner, add-more, open-browser, overwrite md2wiki.yaml
		confirms: []bool{true, false, false, false},
	}
	d.fileExists = func(p string) bool { return p == defaultConfigName }
	d.writeFile = func(p string, data []byte) error { written[p] = data; return nil }

	if err := runInit(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	if _, ok := written["custom.yaml"]; !ok {
		t.Errorf("expected write to custom.yaml, wrote: %v", keys(written))
	}
	if !strings.Contains(out.String(), "md2wiki apply --config custom.yaml") {
		t.Errorf("output missing --config hint:\n%s", out.String())
	}
}

func keys(m map[string][]byte) []string {
	var k []string
	for key := range m {
		k = append(k, key)
	}
	return k
}

func TestRunInit_Aborted(t *testing.T) {
	var out bytes.Buffer
	d := baseDeps(&out)
	d.prompter = &abortingPrompter{}
	wrote := false
	d.writeFile = func(string, []byte) error { wrote = true; return nil }

	if err := runInit(context.Background(), d); err != nil {
		t.Fatalf("aborted wizard should return nil, got %v", err)
	}
	if wrote {
		t.Error("no file should be written on abort")
	}
	if !strings.Contains(out.String(), "Cancelled") {
		t.Errorf("output should mention Cancelled:\n%s", out.String())
	}
}

func TestRunInit_LoopsUntilFreePath(t *testing.T) {
	var out bytes.Buffer
	written := map[string][]byte{}
	d := baseDeps(&out)
	d.prompter = &stubPrompter{
		inputs: []string{
			"docs", "DOCS", "https://x.atlassian.net", "a@b.com", // wizard
			"taken.yaml", // 1st new-path attempt (exists)
			"free.yaml",  // 2nd new-path attempt (free)
		},
		passwords: []string{""},
		selects:   []string{"readme-body", "details"},
		// banner, add-more, open-browser, overwrite md2wiki.yaml, overwrite taken.yaml
		confirms: []bool{true, false, false, false, false},
	}
	exists := map[string]bool{defaultConfigName: true, "taken.yaml": true}
	d.fileExists = func(p string) bool { return exists[p] }
	d.writeFile = func(p string, data []byte) error { written[p] = data; return nil }

	if err := runInit(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	if _, ok := written["free.yaml"]; !ok {
		t.Errorf("expected write to free.yaml, wrote: %v", keys(written))
	}
}

func TestPrintNextSteps_ShellSafeToken(t *testing.T) {
	var out bytes.Buffer
	printNextSteps(&out, wizard.Result{Token: "ab'cd"}, defaultConfigName)
	if !strings.Contains(out.String(), `export MD2WIKI_API_TOKEN='ab'\''cd'`) {
		t.Errorf("token with a single quote must be shell-escaped, got:\n%s", out.String())
	}
}

func TestPrintIntro(t *testing.T) {
	var out bytes.Buffer
	printIntro(&out)
	s := out.String()
	if !strings.Contains(s, "This wizard creates a md2wiki.yaml") {
		t.Errorf("intro missing explanation:\n%s", s)
	}
	if !strings.Contains(s, "never written to the file") {
		t.Errorf("intro should reassure about token:\n%s", s)
	}
}

type abortingPrompter struct{}

func (abortingPrompter) Input(_, _ string, _ func(string) error) (string, error) {
	return "", wizard.ErrAborted
}
func (abortingPrompter) Password(string) (string, error) { return "", wizard.ErrAborted }
func (abortingPrompter) Select(string, []wizard.Choice) (string, error) {
	return "", wizard.ErrAborted
}
func (abortingPrompter) Confirm(string, string, bool) (bool, error) { return false, wizard.ErrAborted }
