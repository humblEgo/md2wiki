// Package cli assembles md2wiki's command-line interface.
package cli

import "github.com/spf13/cobra"

// NewRootCmd creates md2wiki's root cobra command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "md2wiki",
		Short: "Git 마크다운 문서를 Confluence wiki에 단방향 미러링하는 CLI",
		Long: "md2wiki는 repo의 마크다운(SSOT)을 Confluence에 단방향 미러링한다.\n" +
			"repo가 항상 진실이고 Confluence는 사본이다.",
		// When run with no arguments, show the help text instead of doing nothing.
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage: true,
		// The entrypoint (main) is solely responsible for printing errors; this prevents cobra
		// from also printing them, which would otherwise produce duplicate error output.
		SilenceErrors: true,
	}
	cmd.AddCommand(newSyncCmd())
	cmd.AddCommand(newApplyCmd())
	return cmd
}

// Execute runs the root command. It is called from main.
func Execute() error {
	return NewRootCmd().Execute()
}
