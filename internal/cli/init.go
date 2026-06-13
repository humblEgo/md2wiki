package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/humblEgo/md2wiki/internal/confluence"
	"github.com/humblEgo/md2wiki/internal/wizard"
)

const introBanner = `
  ███╗   ███╗██████╗ ██████╗ ██╗    ██╗██╗██╗  ██╗██╗
  ████╗ ████║██╔══██╗╚════██╗██║    ██║██║██║ ██╔╝██║
  ██╔████╔██║██║  ██║ █████╔╝██║ █╗ ██║██║█████╔╝ ██║
  ██║╚██╔╝██║██║  ██║██╔═══╝ ██║███╗██║██║██╔═██╗ ██║
  ██║ ╚═╝ ██║██████╔╝███████╗╚███╔███╔╝██║██║  ██╗██║
  ╚═╝     ╚═╝╚═════╝ ╚══════╝ ╚══╝╚══╝ ╚═╝╚═╝  ╚═╝╚═╝`

// printIntro shows what the tool is and what the wizard will configure, so a first-time
// user understands the flow before answering. Printed to out (not via huh) so it renders
// with full fidelity and is easy to test.
func printIntro(out io.Writer) {
	_, _ = fmt.Fprintln(out, introBanner)
	_, _ = fmt.Fprintln(out, "  Markdown → Confluence, one way.")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "This wizard creates a md2wiki.yaml for you:")
	_, _ = fmt.Fprintln(out, "  • default layout & mermaid rendering modes")
	_, _ = fmt.Fprintln(out, "  • one or more directory → Confluence space mappings")
	_, _ = fmt.Fprintln(out, "  • your Confluence connection (asked last)")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Your API token is never written to the file — it is only used to verify")
	_, _ = fmt.Fprintln(out, "the connection and printed back as a shell export command.")
	_, _ = fmt.Fprintln(out)
}

// initDeps holds runInit's injectable dependencies so the command logic can be tested
// without a TTY, network, or real filesystem writes.
type initDeps struct {
	prompter    wizard.Prompter
	openBrowser func(string) error
	verify      func(ctx context.Context, baseURL, email, token, space string) error
	out         io.Writer
	fileExists  func(string) bool
	writeFile   func(path string, data []byte) error
}

// newInitCmd builds the `md2wiki init` subcommand.
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "대화형 위저드로 md2wiki.yaml을 생성한다",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return errors.New("init must be run in an interactive terminal")
			}
			d := initDeps{
				prompter:    wizard.NewHuhPrompter(),
				openBrowser: wizard.OpenBrowser,
				verify:      defaultVerify,
				out:         cmd.OutOrStdout(),
				fileExists:  func(p string) bool { _, err := os.Stat(p); return err == nil },
				writeFile:   func(p string, data []byte) error { return os.WriteFile(p, data, 0o600) },
			}
			return runInit(cmd.Context(), d)
		},
	}
}

// defaultVerify checks auth + space existence by resolving the space key, reusing the
// existing Confluence client.
func defaultVerify(ctx context.Context, baseURL, email, token, space string) error {
	_, err := confluence.New(baseURL, email, token).SpaceID(ctx, space)
	return err
}

// runInit orchestrates the wizard: collect answers, resolve the output path, write the
// file, verify connections, and print next steps.
func runInit(ctx context.Context, d initDeps) error {
	printIntro(d.out)
	res, err := wizard.Run(d.prompter, d.openBrowser)
	if err != nil {
		if errors.Is(err, wizard.ErrAborted) {
			_, _ = fmt.Fprintln(d.out, "Cancelled.")
			return nil
		}
		return err
	}

	path, err := resolveOutputPath(d)
	if err != nil {
		return err
	}

	data, err := res.File.Marshal()
	if err != nil {
		return err
	}
	if err := d.writeFile(path, data); err != nil {
		return fmt.Errorf("failed to write config file %q: %w", path, err)
	}
	_, _ = fmt.Fprintf(d.out, "\nSaved configuration to %s\n", path)

	if res.Token != "" {
		verifyConnections(ctx, d, res)
	}
	printNextSteps(d.out, res, path)
	return nil
}

// resolveOutputPath returns the path to write. It defaults to md2wiki.yaml; if that
// exists, it asks whether to overwrite, and if not, prompts for a new path (looping
// until a usable path is chosen).
func resolveOutputPath(d initDeps) (string, error) {
	path := defaultConfigName
	for {
		if !d.fileExists(path) {
			return path, nil
		}
		overwrite, err := d.prompter.Confirm(fmt.Sprintf("%s already exists. Overwrite?", path), "", false)
		if err != nil {
			return "", err
		}
		if overwrite {
			return path, nil
		}
		next, err := d.prompter.Input("New config file path", "md2wiki.generated.yaml", nonEmptyPath)
		if err != nil {
			return "", err
		}
		path = next
	}
}

func nonEmptyPath(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("path is required")
	}
	return nil
}

// verifyConnections pings each unique mapping space to confirm auth + existence, printing
// a per-space result. Failures are reported but not fatal — the file is already written.
func verifyConnections(ctx context.Context, d initDeps, res wizard.Result) {
	_, _ = fmt.Fprintln(d.out, "\nVerifying connection...")
	seen := map[string]bool{}
	for _, m := range res.File.Mappings {
		if seen[m.Space] {
			continue
		}
		seen[m.Space] = true
		if err := d.verify(ctx, res.File.BaseURL, res.File.Email, res.Token, m.Space); err != nil {
			_, _ = fmt.Fprintf(d.out, "  [%s] failed: %v\n", m.Space, err)
		} else {
			_, _ = fmt.Fprintf(d.out, "  [%s] OK\n", m.Space)
		}
	}
}

// shellSingleQuote wraps s in single quotes safely for a POSIX shell, escaping any
// embedded single quote as '\'' so the emitted command can't be broken or injected.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// printNextSteps tells the user how to set the token env var and run apply.
func printNextSteps(out io.Writer, res wizard.Result, path string) {
	_, _ = fmt.Fprintln(out, "\nNext steps:")
	tok := res.Token
	if tok == "" {
		tok = "<your-confluence-api-token>"
	}
	_, _ = fmt.Fprintf(out, "  export MD2WIKI_API_TOKEN=%s\n", shellSingleQuote(tok))
	if path == defaultConfigName {
		_, _ = fmt.Fprintln(out, "  md2wiki apply")
	} else {
		_, _ = fmt.Fprintf(out, "  md2wiki apply --config %s\n", path)
	}
}
