package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "revv",
		Short: "Docker test executor for .revv/ configurations",
		Long: `revv executes tests defined in .revv/ inside isolated Docker containers.

Use your IDE (Antigravity, Claude Code, Cursor) to generate and update
test configurations. revv handles the execution.

  IDE:   "revv update"  →  generates .revv/ tests
  IDE:   "revv run"     →  updates tests, then runs revv exec
  CLI:   revv exec      →  executes tests in Docker`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Duration("timeout", 5*time.Minute, "Maximum execution timeout")

	rootCmd.AddCommand(newExecCmd())
	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of revv",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("revv %s (commit: %s, built at: %s)\n", version, commit, date)
		},
	}
}

func Execute() error {
	return NewRootCmd().Execute()
}
