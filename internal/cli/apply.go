package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/humblEgo/md2wiki/internal/config"
	"github.com/humblEgo/md2wiki/internal/confluence"
	"github.com/humblEgo/md2wiki/internal/convert"
	"github.com/humblEgo/md2wiki/internal/walker"
)

// resolvedMapping holds the fully resolved execution parameters for a single mapping.
type resolvedMapping struct {
	source   string // path used at execution time: relative paths are joined onto the config directory, absolute paths are used as-is
	space    string
	rootPage string
	layout   walker.Mode
	mermaid  convert.MermaidMode
	banner   bool
}

// applyPlan holds the connection info plus all mappings for a single apply run.
type applyPlan struct {
	baseURL  string
	email    string
	token    string
	mappings []resolvedMapping
}

// newApplyViper builds a viper bound to MD2WIKI-prefixed environment variables for the apply command.
// Unlike sync's newConfigViper, it sets no defaults: apply's layout/mermaid defaults are not decided
// by viper but by resolveApply itself, via firstNonEmpty(..., "readme-body"/"details"). This keeps
// viper holding only explicitly provided flag/env values so the file vs flag/env precedence stays clear.
func newApplyViper() *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix("MD2WIKI")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	return v
}

func firstNonEmpty(vals ...string) string {
	for _, s := range vals {
		if s != "" {
			return s
		}
	}
	return ""
}

// resolveApply merges the file's global values with the flag/env values (v) into a plan, applying
// the precedence flag/env > file. cfgDir is the directory containing the config file and is used as
// the base for normalizing relative source paths.
func resolveApply(f *config.File, v *viper.Viper, cfgDir string) (applyPlan, error) {
	var plan applyPlan
	plan.baseURL = firstNonEmpty(v.GetString("base-url"), f.BaseURL)
	plan.email = firstNonEmpty(v.GetString("email"), f.Email)
	// For security the token is never stored in the config file: it comes only from the
	// environment (MD2WIKI_API_TOKEN). Unlike baseURL/email, there is deliberately no file fallback.
	plan.token = v.GetString("api-token")

	var missing []string
	if plan.baseURL == "" {
		missing = append(missing, "base-url(--base-url/MD2WIKI_BASE_URL/파일 baseUrl)")
	}
	if plan.email == "" {
		missing = append(missing, "email(--email/MD2WIKI_EMAIL/파일 email)")
	}
	if plan.token == "" {
		missing = append(missing, "MD2WIKI_API_TOKEN env")
	}
	if len(missing) > 0 {
		return plan, fmt.Errorf("필수 설정 누락: %s", strings.Join(missing, ", "))
	}
	if len(f.Mappings) == 0 {
		return plan, fmt.Errorf("mappings가 비어 있습니다: 최소 한 개의 source/space 매핑이 필요합니다")
	}

	for i, m := range f.Mappings {
		if m.Source == "" {
			return plan, fmt.Errorf("mappings[%d]: source가 비어 있습니다", i)
		}
		if m.Space == "" {
			return plan, fmt.Errorf("mappings[%d] (%s): space가 비어 있습니다", i, m.Source)
		}
		layout, err := parseLayout(firstNonEmpty(m.LayoutMode, f.LayoutMode, "readme-body"))
		if err != nil {
			return plan, fmt.Errorf("mappings[%d] (%s): %w", i, m.Source, err)
		}
		mmode, err := parseMermaidMode(firstNonEmpty(m.MermaidMode, f.MermaidMode, "details"))
		if err != nil {
			return plan, fmt.Errorf("mappings[%d] (%s): %w", i, m.Source, err)
		}
		src := m.Source
		if !filepath.IsAbs(src) {
			src = filepath.Join(cfgDir, src)
		}
		banner := true
		if f.Banner != nil {
			banner = *f.Banner
		}
		if m.Banner != nil {
			banner = *m.Banner
		}
		plan.mappings = append(plan.mappings, resolvedMapping{
			source: src, space: m.Space, rootPage: m.RootPage, layout: layout, mermaid: mmode, banner: banner,
		})
	}
	return plan, nil
}

const defaultConfigName = "md2wiki.yaml"

// newApplyCmd builds the `md2wiki apply` subcommand.
func newApplyCmd() *cobra.Command {
	v := newApplyViper()
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "md2wiki.yaml에 선언된 매핑들을 일괄 미러링한다",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			if path == "" {
				path = defaultConfigName
			}
			f, err := config.Load(path)
			if err != nil {
				return err
			}
			plan, err := resolveApply(f, v, filepath.Dir(path))
			if err != nil {
				return err
			}
			return runApply(cmd, plan)
		},
	}
	cmd.Flags().String("config", "", "설정 파일 경로(기본: 현재 디렉토리의 md2wiki.yaml)")
	cmd.Flags().String("base-url", "", "Confluence base URL — 파일 baseUrl 재정의 [env MD2WIKI_BASE_URL]")
	cmd.Flags().String("email", "", "Confluence 계정 이메일 — 파일 email 재정의 [env MD2WIKI_EMAIL]")
	_ = v.BindPFlags(cmd.Flags())
	return cmd
}

// runApply synchronizes the plan's mappings sequentially, fail-fast: the first error aborts the
// whole run so a partial failure does not silently leave later mappings unmirrored.
func runApply(cmd *cobra.Command, plan applyPlan) error {
	client := confluence.New(plan.baseURL, plan.email, plan.token)
	for _, m := range plan.mappings {
		res, err := runPipeline(cmd.Context(), client, m.source, m.space, m.rootPage, m.layout, m.mermaid, m.banner)
		if err != nil {
			return fmt.Errorf("apply [%s] %s: %w", m.space, m.source, err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: created=%d updated=%d skipped=%d\n",
			m.space, m.source, res.Created, res.Updated, res.Skipped)
	}
	return nil
}
