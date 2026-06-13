# revv

**IDE-first QA automation for open-source repos.**

revv replaces manual QA by generating and running automated tests for every code change. Your IDE's LLM handles the intelligence — revv handles the execution.

## How It Works

```
┌──────────────────────────────────┐
│ Your IDE (Antigravity / Claude   │
│ Code / Cursor / Codex)           │
│                                  │
│  "revv update"  →  generates     │
│                    .revv/ tests  │
│                                  │
│  "revv run"     →  updates tests │
│                    then executes │
│                    revv exec     │
└──────────────┬───────────────────┘
               │
        ┌──────▼──────┐
        │ revv exec   │
        │ (Go binary) │
        │ Docker      │
        │ sandboxed   │
        │ execution   │
        └─────────────┘
```

**No API key needed.** Your IDE already has an LLM — revv uses it.

## Quick Start

### 1. Install

```bash
go install github.com/vipinsingh/revv/cmd/revv@latest
```

### 2. Generate Tests

Open your IDE and say:

> **"revv update"**

The IDE reads your codebase, analyzes your changes, and generates a `.revv/` directory with tests:

```
.revv/
├── Dockerfile           ← Docker sandbox for this project
├── helpers/
│   └── assert.sh        ← shared helper scripts
├── build/
│   └── compile_check/
│       └── test.md      ← "does it compile?"
├── cli_sanity/
│   └── version_check/
│       └── test.md      ← "does --version work?"
└── manual/
    └── dark_mode/
        └── test.md      ← "visual check (human steps)"
```

### 3. Run Tests

> **"revv run"**

The IDE first checks if your changes need new tests (runs `revv update`), then executes all tests:

```
$ revv exec --verbose

  ✓ build/compile_check        blocking  PASS  (0.5s)
  ✓ cli_sanity/version_check   blocking  PASS  (0.2s)
  ✗ cli_sanity/timeout_flag    blocking  FAIL  (0.1s)

Results: 2 passed, 1 failed
```

The IDE then analyzes failures and tells you exactly what went wrong and how to fix it.

### 4. CI

In GitHub Actions, just run the executor — no LLM needed:

```yaml
- run: revv exec
```

Tests are already generated and committed. CI just runs them.

## Test Format

Each test is a `test.md` file:

```markdown
## Description
Verify the CLI binary compiles and is executable.

## Priority
blocking

## Commands
```bash
make build
test -x ./bin/myapp || (echo "FAIL" && exit 1)
echo "PASS"
```

## Expected Output
Exit code 0.
```

- **Priority**: `blocking` (merge-gating) or `warning` (advisory)
- **Commands**: Real shell commands executed inside Docker
- **Steps**: For manual tests that can't be automated

## CLI Reference

```
revv exec [flags]
  --category string   Run only a specific category
  --test string       Run a single test by path
  --json              Output results as JSON
  --verbose           Show detailed output
  --timeout duration  Global timeout (default: 5m)
```

## Architecture

```
cmd/revv/          ← binary entrypoint
internal/
├── cli/           ← cobra commands (exec, version)
├── runner/        ← test discovery, parsing, parallel execution
├── sandbox/       ← Docker container lifecycle
└── git/           ← git utilities
```

The binary has **zero LLM dependencies**. All intelligence lives in the IDE skills (`AGENTS.md`, `CLAUDE.md`).

## License

MIT
