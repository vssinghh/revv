# Project: revv

LLM-powered PR review automation tool for open-source maintainers. Implements the initialization phase (`revv init`) to scaffold repository configuration.

## Architecture
- **CLI Commands (`internal/cli/`)**: Cobra routing, flag parsing (`--model`), config validation (`GEMINI_API_KEY`), and output reporting.
- **Context Collection (`internal/context/`)**: Scan workspace, read files (`CONTRIBUTING.md`, `README.md`, `Makefile`, `Dockerfile`), and serialize repository context.
- **LLM Client (`internal/llm/`)**: Call Gemini via the `adk-go` SDK (using model-agnostic agent interface) to generate `.revv/Dockerfile`, global helpers, and categorized test folders containing `test.md` and category-specific helper scripts.
- **Git Integration (`internal/git/`)**: Create local git branch `revv/init`, stage `.revv/` contents, commit, and output next steps (`git push`).

## Code Layout
- `cmd/revv/main.go` - CLI Entrypoint
- `internal/cli/` - Cobra subcommands (`root.go`, `init.go`)
- `internal/context/` - Repository scanner and context reader
- `internal/llm/` - SDK integration & prompt construction
- `internal/git/` - Git automation logic
- `Makefile` - Project automation (`build`, `test`, `clean`)

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|---|---|-------------|--------|
| 1 | Repository Scaffold | Initialize git, .gitignore, go.mod, LICENSE, README.md, Makefile | none | IN_PROGRESS |
| 2 | E2E Test Infra | Implement Tier 1-4 tests and test runner, publish TEST_READY.md | M1 | PLANNED |
| 3 | CLI Base & Routing | CLI subcommand structure (Cobra), help commands, flag/key checking | M1, M2 | PLANNED |
| 4 | Context & Gemini SDK | Read project files, construct prompt, invoke Gemini via `adk-go`, write `.revv/` | M3 | PLANNED |
| 5 | Git Operations | Branch creation `revv/init`, stage/commit files, print next steps | M4 | PLANNED |
| 6 | E2E Pass & Audit | Complete E2E pass, run adversarial hardening (Tier 5), Forensic Audit | M5 | PLANNED |

## Interface Contracts
### `internal/cli` ↔ `internal/context`
- Function: `ReadRepositoryContext(dir string) (map[string]string, error)`
- Purpose: CLI invokes context collection on the target directory, getting a map of filename to content.

### `internal/cli` ↔ `internal/llm`
- Function: `GenerateConfig(ctx context.Context, modelName string, repoContext map[string]string) (*ConfigOutput, error)`
- Types:
  ```go
  type ConfigOutput struct {
      Dockerfile string
      Helpers    map[string]string // relative path -> content
      Tests      []TestInfo
  }
  type TestInfo struct {
      Category string
      Name     string
      TestMD   string
      Helpers  map[string]string // relative path -> content
  }
  ```

### `internal/cli` ↔ `internal/git`
- Function: `PrepareBranchAndCommit(dir string, files []string) error`
- Purpose: CLI commands git package to check out branch `revv/init`, stage files, and commit.
