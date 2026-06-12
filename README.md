# revv

LLM-powered PR review automation for open-source maintainers. Revv reads your repo, generates targeted tests using Gemini, and runs them in isolated Docker sandboxes — one container per test, fully parallel.

```bash
# Onboard a repo
export GEMINI_API_KEY="your-key"
revv init

# Run generated tests
revv run --verbose
```

---

## Why Revv

Open-source maintainers review dozens of PRs weekly. Most lack automated QA beyond CI unit tests. Revv acts as an AI QA engineer:

1. **`revv init`** — Reads your codebase (README, Makefile, go.mod, source files) and asks Gemini to generate a Dockerfile + test suite tailored to your project
2. **`revv run`** — Builds a Docker image, spins up one container per test (full isolation), runs all tests in parallel, reports results

No mocks. No fake backends. Tests exercise real product behavior in a sandboxed environment.

---

## Installation

### Prerequisites
- **Go** 1.26.4+ ([download](https://go.dev/dl/))
- **Docker** — Docker Desktop, Colima, or Rancher Desktop. If Docker is missing, revv will attempt to install Colima + Docker CLI automatically.
- **Gemini API Key** — Get one from [Google AI Studio](https://aistudio.google.com/app/apikey)

### Build from Source

```bash
git clone https://github.com/vssinghh/revv.git
cd revv
make build
```

Produces `bin/revv`. Optionally add to your PATH:

```bash
export PATH="$PWD/bin:$PATH"
```

---

## Usage

### `revv init`

Generates a `.revv/` directory with a Dockerfile and test suite for your project.

```bash
cd your-repo
export GEMINI_API_KEY="your-key"
revv init
```

**What it does:**
1. Collects context files (README, Makefile, go.mod, source code, existing tests)
2. Sends them to Gemini 3.5 Flash with a structured prompt
3. Parses the LLM response into files
4. Creates a `revv/init` branch with the `.revv/` directory committed
5. Prints instructions to push and open a PR

**Flags:**
- `--model <model>` — Gemini model to use (default: `gemini-3.5-flash`)
- `--verbose` — Show detailed output including collected files

**Generated structure:**
```
.revv/
├── Dockerfile                    # Build environment (correct language version, deps, pre-built binary)
├── helpers/                      # Shared scripts across categories
│   └── assert.sh
├── build/
│   └── compile_check/
│       └── test.md               # Blocking: verifies compilation
├── error_handling/
│   ├── missing_api_key_init/
│   │   └── test.md               # Blocking: tests missing key error
│   └── run_missing_revv_dir/
│       └── test.md               # Blocking: tests missing config error
├── cli_sanity/
│   └── version_check/
│       └── test.md               # Blocking: version command works
└── visual/
    └── cli_layout/
        └── test.md               # Warning: help output formatting
```

### `revv run`

Executes all tests from `.revv/` in isolated Docker containers.

```bash
revv run --verbose
```

**What it does:**
1. Checks Docker availability (auto-installs Colima if needed)
2. Scans test.md files for `$VARIABLE` references and auto-forwards matching host env vars
3. Builds a Docker image from `.revv/Dockerfile` (cached — only rebuilds on changes)
4. Runs every test in its own fresh container — full filesystem isolation, no state leakage
5. All tests run in parallel via goroutines
6. Reports pass/fail with timing for each test

**Flags:**
- `--category <name>` — Run only tests in a specific category (e.g., `build`)
- `--test <category/name>` — Run a single test (e.g., `error_handling/missing_api_key_init`)
- `--timeout <duration>` — Global timeout (default: `10m`)
- `--verbose` — Show full output for failed tests

**Example output:**
```
Checking Docker availability...

Environment variables detected from tests:
  GEMINI_API_KEY                 ✓ (host)

Building sandbox from .revv/Dockerfile...
Running tests (parallel, isolated containers):
  ✓ build/compile_check                      blocking   PASS   (0.3s)
  ✓ cli_sanity/version_check                 blocking   PASS   (0.2s)
  ✓ error_handling/missing_api_key_init      blocking   PASS   (0.2s)
  ✓ error_handling/run_missing_revv_dir      blocking   PASS   (0.2s)
  ✓ visual/cli_layout                        warning    PASS   (0.2s)

Results: 5 passed, 0 failed, 0 skipped
Blocking: 4/4 passed ✓

Sandbox cleaned up.
```

### `revv version`

Prints version, commit hash, and build date.

---

## How Tests Work

### test.md Format

Each test is a markdown file with structured sections:

```markdown
## Description
What this test verifies and why it matters.

## Priority
blocking | warning

## Commands
\`\`\`bash
./bin/revv version | grep -q "revv" || (echo "FAIL: no version output" && exit 1)
echo "PASS: version check"
\`\`\`

## Expected Output
Exit code 0. Output contains "PASS: version check".
```

### Priorities
- **blocking** — PR cannot merge if this test fails
- **warning** — Informational, does not block merge

The LLM may output synonyms (high, critical, p0). The parser normalizes them:
- `high`, `critical`, `p0`, `must-pass` → **blocking**
- `low`, `medium`, `informational`, `p1`, `p2` → **warning**

### Categories
The LLM organizes tests into categories based on what they test:
- `build` — Compilation, binary output
- `cli_sanity` — CLI commands work as documented
- `error_handling` — Graceful failures with actionable messages
- `visual` — UI rendering, help layout, formatting (always generated)

### Environment Variables

Tests reference env vars with standard `$VARIABLE` syntax. The runner:
1. Scans all test.md files for `$VAR` patterns
2. Filters out shell builtins (`HOME`, `PATH`, `PWD`, etc.)
3. Checks host environment and `.env` files
4. Passes matching variables into containers automatically

No configuration needed — if your test uses `$GEMINI_API_KEY` and it's set on your machine, it gets forwarded.

---

## Architecture

```
cmd/revv/main.go              → CLI entrypoint
internal/
├── cli/                       → Cobra commands (init, run, version)
│   ├── init.go                → revv init implementation
│   ├── run.go                 → revv run implementation
│   └── root.go                → Root command, global flags
├── adk/
│   └── client.go              → LLM prompt construction + Gemini API call
├── llm/
│   └── llm.go                 → JSON schema, response parsing, mock mode
├── runner/
│   ├── runner.go              → Test discovery + parallel execution
│   ├── parser.go              → test.md parsing + priority normalization
│   └── env.go                 → Environment variable detection
├── sandbox/
│   ├── sandbox.go             → Docker container lifecycle (per-test isolation)
│   └── install.go             → Auto-install Colima + Docker CLI
├── context/
│   ├── context.go             → Repository file collection for LLM
│   └── revv.go                → Existing .revv/ config reading
└── git/
    └── git.go                 → Branch creation, staging, committing
tests/
└── e2e/
    └── e2e_test.go            → End-to-end CLI tests
```

### Key Design Decisions

**One container per test.** Each `Exec()` call creates a fresh Docker container from the cached image. Tests cannot leak state to each other. Container startup from a cached image is ~100ms.

**Parallel execution.** All tests run concurrently via goroutines with `sync.WaitGroup`. Results are collected into an indexed slice (no channels needed). Wall time ≈ slowest test, not sum of all tests.

**Pre-built binaries.** The Dockerfile includes `RUN make build` so the compiled binary is baked into the image. Tests start instantly instead of recompiling.

**No mocks in generated tests.** The LLM is explicitly instructed not to use mock modes (`REVV_MOCK_LLM`, `TEST_MODE`, etc.). Generated tests exercise real product behavior. If a feature needs an external service unavailable in the sandbox, tests validate the pre-condition error path instead.

**Docker socket discovery.** Supports multiple Docker runtimes without configuration:
- Docker Desktop: `/var/run/docker.sock`
- Colima: `~/.colima/default/docker.sock`
- Rancher Desktop: `~/.rd/docker.sock`

**Auto-install.** If Docker is not available, `revv run` installs Colima + Docker CLI via Homebrew (macOS) or apt (Linux) with user notification.

---

## Development

### Building and Testing

```bash
make build              # Compile binary to bin/revv
make test               # Run all Go tests
make clean              # Remove build artifacts
```

### Adding a CLI Command

1. Create `internal/cli/<command>.go` with `newXxxCmd()` returning a `*cobra.Command`
2. Register in `internal/cli/root.go` via `rootCmd.AddCommand()`
3. Add E2E tests in `tests/e2e/e2e_test.go`

### Modifying the LLM Prompt

The prompt is built programmatically in `internal/adk/client.go` → `BuildPrompt()`. It includes:
- Repository context (files, existing tests, file tree)
- JSON schema for structured output
- Rules for test generation (no mocks, pre-build Dockerfile, mandatory visual category)

Test changes by running `revv init` with a real API key and inspecting the output.

### Mock LLM

Set `REVV_MOCK_LLM=true` for Go unit tests and E2E tests. This avoids hitting the Gemini API during `go test`. The mock returns a fixed JSON response with realistic test data.

**Important:** Mock mode is for internal development tests only. Generated `.revv/` tests must never use mocks.

### Validation Checklist

Before submitting a PR:

1. `go build ./cmd/revv` — must compile
2. `go test ./...` — all unit + E2E tests pass
3. `GEMINI_API_KEY=<key> ./bin/revv init` — generates valid `.revv/` config
4. `./bin/revv run --verbose` — generated tests pass in Docker sandbox

---

## License

MIT License. See [LICENSE](LICENSE) for details.
