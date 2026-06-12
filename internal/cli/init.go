package cli

import (
	stdctx "context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	repoctx "github.com/vipinsingh/revv/internal/context"
	"github.com/vipinsingh/revv/internal/git"
	"github.com/vipinsingh/revv/internal/llm"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize revv configuration in the repository",
		Long:  `Scaffolds repository configuration using Gemini and prepares a git branch with the results.`,
		Args:  cobra.NoArgs,
		RunE:  runInit,
	}

	cmd.Flags().String("model", "gemini-3.5-flash", "Gemini model to use for generation")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	modelName, _ := cmd.Flags().GetString("model")

	if modelName == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	if verbose {
		fmt.Println("Collecting repository context...")
	}
	repoCtx, err := repoctx.ReadRepositoryContext(cwd)
	if err != nil {
		return fmt.Errorf("failed to read repository context: %w", err)
	}

	if verbose {
		for k := range repoCtx {
			fmt.Printf("Collected context file: %s\n", k)
		}
		fmt.Printf("Invoking Gemini (%s) to generate configuration...\n", modelName)
	}
	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), timeout)
	defer cancel()

	configOutput, err := llm.GenerateConfig(ctx, modelName, repoCtx)
	if err != nil {
		return fmt.Errorf("failed to generate configuration: %w", err)
	}

	if verbose {
		fmt.Println("Writing generated files to .revv/...")
	}

	revvDir := ".revv"
	if err := os.MkdirAll(revvDir, 0755); err != nil {
		return fmt.Errorf("failed to create .revv directory: %w", err)
	}

	// Always ensure a manual category (.revv/manual/) is generated
	manualDir := filepath.Join(revvDir, "manual")
	if err := os.MkdirAll(manualDir, 0755); err != nil {
		return fmt.Errorf("failed to create manual category directory: %w", err)
	}

	var writtenFiles []string

	dockerfilePath := filepath.Join(revvDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(configOutput.Dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}
	writtenFiles = append(writtenFiles, dockerfilePath)

	for path, content := range configOutput.Helpers {
		var fullPath string
		if strings.HasPrefix(path, "helpers/") {
			fullPath = filepath.Join(revvDir, path)
		} else {
			fullPath = filepath.Join(revvDir, "helpers", path)
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for helper %s: %w", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write helper %s: %w", path, err)
		}
		writtenFiles = append(writtenFiles, fullPath)
	}

	for _, test := range configOutput.Tests {
		categoryDir := filepath.Join(revvDir, test.Category)
		if err := os.MkdirAll(categoryDir, 0755); err != nil {
			return fmt.Errorf("failed to create category directory %s: %w", test.Category, err)
		}

		testDir := filepath.Join(categoryDir, test.Name)
		if err := os.MkdirAll(testDir, 0755); err != nil {
			return fmt.Errorf("failed to create test directory %s: %w", test.Name, err)
		}

		testMDPath := filepath.Join(testDir, "test.md")
		if err := os.WriteFile(testMDPath, []byte(test.TestMD), 0644); err != nil {
			return fmt.Errorf("failed to write test %s: %w", test.Name, err)
		}
		writtenFiles = append(writtenFiles, testMDPath)

		for path, content := range test.Helpers {
			var helperPath string
			if strings.HasPrefix(path, "helpers/") {
				helperPath = filepath.Join(categoryDir, path)
			} else {
				helperPath = filepath.Join(categoryDir, "helpers", path)
			}
			if err := os.MkdirAll(filepath.Dir(helperPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for test helper %s: %w", path, err)
			}
			if err := os.WriteFile(helperPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write test helper %s: %w", path, err)
			}
			writtenFiles = append(writtenFiles, helperPath)
		}
	}

	if verbose {
		fmt.Println("Staging and committing files to git...")
	}
	if err := git.PrepareBranchAndCommit(cwd, writtenFiles); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	fmt.Println("Initialization complete! Proposed configuration committed to branch 'revv/init'.")
	fmt.Println("Run 'git push origin revv/init' to submit it for review.")

	return nil
}
