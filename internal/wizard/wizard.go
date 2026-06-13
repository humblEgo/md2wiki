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
func Run(p Prompter, openBrowser func(string) error) (Result, error) {
	var res Result

	baseURL, err := p.Input("Confluence base URL", "https://your-team.atlassian.net", validateURL)
	if err != nil {
		return res, err
	}
	res.File.BaseURL = baseURL

	email, err := p.Input("Confluence 계정 이메일", "you@your-team.com", validateNonEmpty)
	if err != nil {
		return res, err
	}
	res.File.Email = email

	openIt, err := p.Confirm("API 토큰 발급 페이지를 브라우저로 열까요?", true)
	if err != nil {
		return res, err
	}
	if openIt {
		_ = openBrowser(TokenPageURL) // 실패해도 흐름은 계속한다(끝에서 URL을 다시 안내).
	}
	token, err := p.Password("API 토큰 (선택 — 비우면 나중에 환경변수로 설정)")
	if err != nil {
		return res, err
	}
	res.Token = token

	layout, err := p.Select("기본 layout 모드", []string{"readme-body", "mirror"})
	if err != nil {
		return res, err
	}
	res.File.LayoutMode = layout

	mermaid, err := p.Select("기본 mermaid 모드", []string{"details", "render", "raw"})
	if err != nil {
		return res, err
	}
	res.File.MermaidMode = mermaid

	banner, err := p.Confirm("미러링 안내 배너를 기본으로 켤까요?", true)
	if err != nil {
		return res, err
	}
	res.File.Banner = &banner

	for {
		m, err := promptMapping(p)
		if err != nil {
			return res, err
		}
		res.File.Mappings = append(res.File.Mappings, m)
		more, err := p.Confirm("매핑을 더 추가할까요?", false)
		if err != nil {
			return res, err
		}
		if !more {
			break
		}
	}

	return res, nil
}

func promptMapping(p Prompter) (config.Mapping, error) {
	var m config.Mapping
	source, err := p.Input("매핑 소스 디렉토리 (md2wiki.yaml 기준 상대경로)", "docs", validateNonEmpty)
	if err != nil {
		return m, err
	}
	m.Source = source
	space, err := p.Input("대상 스페이스 키", "DOCS", validateNonEmpty)
	if err != nil {
		return m, err
	}
	m.Space = space
	rootPage, err := p.Input("부모 페이지 ID (선택, 비우면 스페이스 최상위)", "", nil)
	if err != nil {
		return m, err
	}
	m.RootPage = rootPage
	return m, nil
}
