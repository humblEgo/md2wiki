package wizard

import "github.com/humblEgo/md2wiki/internal/config"

// TokenPageURL is Atlassian's API-token management page, opened during the wizard.
const TokenPageURL = "https://id.atlassian.com/manage-profile/security/api-tokens"

// Result is the wizard's output: the config to write, plus the token (kept in memory
// only — never serialized into the file).
type Result struct {
	File  config.File
	Token string
}

// Run drives the interactive flow using p for prompts and openBrowser to launch the
// token page. It returns ErrAborted (via p) if the user cancels.
//
// Connection details (base URL, email, token) are asked LAST so a user without
// credentials ready can still walk through the rest of the setup.
func Run(p Prompter, openBrowser func(string) error) (Result, error) {
	var res Result

	// Defaults first — no credentials required.
	layout, err := p.Select("Default layout mode", []Choice{
		{Value: "readme-body", Desc: "A folder's README.md becomes the folder page's body; other .md files become child pages. (default)"},
		{Value: "mirror", Desc: "1:1 reflection of the filesystem — README.md becomes an ordinary page too."},
	})
	if err != nil {
		return res, err
	}
	res.File.LayoutMode = layout

	mermaid, err := p.Select("Default mermaid mode", []Choice{
		{Value: "details", Desc: "Rendered image plus the original source in a collapsible region. (default)"},
		{Value: "render", Desc: "Rendered PNG image only."},
		{Value: "raw", Desc: "Original mermaid as a code block; works without any external tooling."},
	})
	if err != nil {
		return res, err
	}
	res.File.MermaidMode = mermaid

	banner, err := p.Confirm("Enable the mirror notice banner by default?", true)
	if err != nil {
		return res, err
	}
	res.File.Banner = &banner

	// Mappings — the heart of the config.
	for {
		m, err := promptMapping(p)
		if err != nil {
			return res, err
		}
		res.File.Mappings = append(res.File.Mappings, m)
		more, err := p.Confirm("Add another mapping?", false)
		if err != nil {
			return res, err
		}
		if !more {
			break
		}
	}

	// Connection details last.
	baseURL, err := p.Input("Confluence base URL", "https://your-team.atlassian.net", validateURL)
	if err != nil {
		return res, err
	}
	res.File.BaseURL = baseURL

	email, err := p.Input("Confluence account email", "you@your-team.com", validateNonEmpty)
	if err != nil {
		return res, err
	}
	res.File.Email = email

	openIt, err := p.Confirm("Open the API token page in your browser?", true)
	if err != nil {
		return res, err
	}
	if openIt {
		_ = openBrowser(TokenPageURL) // 실패해도 흐름은 계속한다(호출부가 끝에서 URL을 다시 안내).
	}
	token, err := p.Password("Confluence API token (optional — leave empty to set it later via env)")
	if err != nil {
		return res, err
	}
	res.Token = token

	return res, nil
}

func promptMapping(p Prompter) (config.Mapping, error) {
	var m config.Mapping
	source, err := p.Input("Source directory to mirror (relative to md2wiki.yaml)", "docs", validateNonEmpty)
	if err != nil {
		return m, err
	}
	m.Source = source
	space, err := p.Input("Target Confluence space key", "DOCS", validateNonEmpty)
	if err != nil {
		return m, err
	}
	m.Space = space
	rootPage, err := p.Input("Parent page ID (optional — empty = space root)", "", nil)
	if err != nil {
		return m, err
	}
	m.RootPage = rootPage
	return m, nil
}
