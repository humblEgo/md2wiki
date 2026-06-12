package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/humblEgo/md2wiki/internal/confluence"
	"github.com/humblEgo/md2wiki/internal/convert"
	"github.com/humblEgo/md2wiki/internal/convert/mermaid"
	"github.com/humblEgo/md2wiki/internal/sync"
	"github.com/humblEgo/md2wiki/internal/title"
	"github.com/humblEgo/md2wiki/internal/walker"
)

// runConfig holds the fully resolved settings needed to run a sync.
type runConfig struct {
	dir         string
	baseURL     string
	email       string
	token       string
	space       string
	parentID    string
	layout      walker.Mode
	mermaidMode convert.MermaidMode
	banner      bool
}

// newConfigViper builds a viper bound to MD2WIKI-prefixed environment variables, with the sync
// command's defaults pre-set so that an unspecified flag/env falls back to a sensible value.
func newConfigViper() *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix("MD2WIKI")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	v.SetDefault("layout-mode", "readme-body")
	v.SetDefault("mermaid-mode", "details")
	v.SetDefault("banner", true)
	return v
}

// loadConfig reads values from viper, then validates and parses them. dir is the positional argument.
func loadConfig(v *viper.Viper, dir string) (runConfig, error) {
	cfg := runConfig{
		dir:      dir,
		baseURL:  v.GetString("base-url"),
		email:    v.GetString("email"),
		space:    v.GetString("space"),
		token:    v.GetString("api-token"),
		parentID: v.GetString("parent-id"),
	}
	var missing []string
	if cfg.baseURL == "" {
		missing = append(missing, "--base-url")
	}
	if cfg.email == "" {
		missing = append(missing, "--email")
	}
	if cfg.space == "" {
		missing = append(missing, "--space")
	}
	if cfg.token == "" {
		missing = append(missing, "MD2WIKI_API_TOKEN env")
	}
	if len(missing) > 0 {
		return cfg, fmt.Errorf("필수 설정 누락: %s", strings.Join(missing, ", "))
	}

	layout, err := parseLayout(v.GetString("layout-mode"))
	if err != nil {
		return cfg, err
	}
	mode, err := parseMermaidMode(v.GetString("mermaid-mode"))
	if err != nil {
		return cfg, err
	}
	cfg.layout = layout
	cfg.mermaidMode = mode
	cfg.banner = v.GetBool("banner")
	return cfg, nil
}

func parseLayout(s string) (walker.Mode, error) {
	switch s {
	case "readme-body":
		return walker.ModeReadmeBody, nil
	case "mirror":
		return walker.ModeMirror, nil
	default:
		return 0, fmt.Errorf("알 수 없는 layout-mode %q (readme-body|mirror 중 하나)", s)
	}
}

func parseMermaidMode(s string) (convert.MermaidMode, error) {
	switch s {
	case "details":
		return convert.MermaidDetails, nil
	case "render":
		return convert.MermaidRender, nil
	case "raw":
		return convert.MermaidRaw, nil
	default:
		return 0, fmt.Errorf("알 수 없는 mermaid-mode %q (details|render|raw 중 하나)", s)
	}
}

// newSyncCmd builds the `md2wiki sync <dir>` subcommand.
func newSyncCmd() *cobra.Command {
	v := newConfigViper()
	cmd := &cobra.Command{
		Use:   "sync <dir>",
		Short: "디렉토리의 마크다운을 Confluence에 미러링한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(v, args[0])
			if err != nil {
				return err
			}
			return runSync(cmd, cfg)
		},
	}
	cmd.Flags().String("base-url", "", "Confluence base URL, 예: https://acme.atlassian.net [env MD2WIKI_BASE_URL]")
	cmd.Flags().String("email", "", "Confluence 계정 이메일 [env MD2WIKI_EMAIL]")
	cmd.Flags().String("space", "", "대상 스페이스 키 [env MD2WIKI_SPACE]")
	cmd.Flags().String("parent-id", "", "부모 페이지 ID(선택). 비우면 스페이스 최상위 [env MD2WIKI_PARENT_ID]")
	cmd.Flags().String("layout-mode", "readme-body", "레이아웃: readme-body | mirror [env MD2WIKI_LAYOUT_MODE]")
	cmd.Flags().String("mermaid-mode", "details", "mermaid: details | render | raw [env MD2WIKI_MERMAID_MODE]")
	cmd.Flags().Bool("banner", true, "미러링 안내 배너를 페이지 맨 위에 삽입(기본 켜짐). 끄려면 --banner=false [env MD2WIKI_BANNER]")
	_ = v.BindPFlags(cmd.Flags())
	return cmd
}

// runPipeline is the shared pipeline that processes one directory: walk -> resolve titles -> synchronize.
// The client is created and injected by the caller so it can be reused across multiple mappings.
func runPipeline(ctx context.Context, client sync.API, dir, space, rootPage string, layout walker.Mode, mmode convert.MermaidMode, banner bool) (sync.Result, error) {
	tr, err := walker.Walk(dir, layout)
	if err != nil {
		return sync.Result{}, err
	}
	if err := title.Resolve(dir, tr); err != nil {
		return sync.Result{}, err
	}
	return sync.Run(ctx, client, tr, sync.Config{
		Root:            dir,
		SpaceKey:        space,
		RootParentID:    rootPage,
		MermaidMode:     mmode,
		MermaidRenderer: mermaid.New(),
		Banner:          banner,
	})
}

// runSync executes the single-directory sync command.
func runSync(cmd *cobra.Command, cfg runConfig) error {
	client := confluence.New(cfg.baseURL, cfg.email, cfg.token)
	res, err := runPipeline(cmd.Context(), client, cfg.dir, cfg.space, cfg.parentID, cfg.layout, cfg.mermaidMode, cfg.banner)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "created=%d updated=%d skipped=%d\n", res.Created, res.Updated, res.Skipped)
	return nil
}
