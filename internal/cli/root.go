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
		Short: "LLM-powered PR review automation tool",
		Long:  `revv is an LLM-powered PR review automation tool for open-source maintainers.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Duration("timeout", 5*time.Minute, "Maximum execution timeout")

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newRunCmd())
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

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current revv configuration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("revv status: configured")
			return nil
		},
	}
}
