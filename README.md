# revv

**Automated QA for open-source repos, powered by your IDE.**

revv replaces manual PR review with automated tests that run inside Docker containers. Your IDE's built-in LLM generates tests, executes them, analyzes failures, and even runs browser-based UI tests — all without an API key.

## How It Works

```
Contributor says "revv run" in their IDE
        │
        ▼
┌─────────────────────────────────┐
│ IDE reads AGENTS.md             │
│ Fetches latest skill from       │
│ github.com/vssinghh/revv        │
│                                 │
│ 1. Checks if tests need updates │
│ 2. Runs automated tests (Docker)│
│ 3. Runs browser tests (Chrome)  │
│ 4. Analyzes failures            │
│ 5. Reports results              │
└─────────────────────────────────┘
```

**No API key. No binary install. No setup.** Contributors just clone and say "revv run".

## For Maintainers

### Setting up revv in your repo

1. Open your IDE (Antigravity, Claude Code, Cursor, Codex)
2. Say: **"revv update"**
3. The IDE generates:
   - `.revv/Dockerfile` — Docker sandbox for your project
   - `.revv/<category>/<test>/test.md` — test definitions
   - `AGENTS.md` — pointer file so contributors can use revv too
4. Review the generated files, commit, and push

That's it. Your repo now has automated QA.

### What gets generated

```
your-repo/
├── AGENTS.md              ← pointer to revv skills (never goes stale)
└── .revv/
    ├── Dockerfile         ← Docker sandbox (pre-builds your project)
    ├── helpers/
    │   └── assert.sh
    ├── build/
    │   └── compile_check/
    │       └── test.md    ← "does it compile?"
    ├── unit/
    │   └── parser_test/
    │       └── test.md    ← "does the parser handle edge cases?"
    └── manual/
        └── login_flow/
            └── test.md    ← "does the login UI work?" (browser test)
```

### How contributors use it

Contributors just clone your repo and say "revv run" in their IDE. The `AGENTS.md` in your repo points to the latest revv skills on GitHub, so instructions are always up to date.

The contributor's IDE:
1. Reads `AGENTS.md`
2. Fetches the latest skill instructions from this repo
3. Checks if the contributor's changes need new tests
4. Runs all tests in Docker containers
5. Runs browser-based tests via Chrome DevTools
6. Analyzes any failures and suggests fixes

### CI Integration

For GitHub Actions, use `revv exec` (the optional Go binary):

```yaml
name: revv
on: [pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go install github.com/vssinghh/revv/cmd/revv@latest
      - run: revv exec --json
```

Or without Go — use the `revv exec` self-build:

```yaml
      - run: docker build -t revv-sandbox -f .revv/Dockerfile .
      - run: |
          # Build revv from source and run
          go build -o /tmp/revv github.com/vssinghh/revv/cmd/revv
          /tmp/revv exec --json
```

## Test Format

Each test is a `test.md` file with markdown headings.

### Automated tests (run in Docker)

A test.md with `## Commands` runs inside a Docker container:

> **## Description**
> Verify the CLI binary compiles and is executable.
>
> **## Priority**
> `blocking`
>
> **## Commands**
> ```bash
> make build
> test -x ./bin/myapp || (echo "FAIL" && exit 1)
> echo "PASS"
> ```
>
> **## Expected Output**
> Exit code 0.

### Browser tests (run by IDE)

A test.md with `## Steps` is executed by the IDE's browser automation:

> **## Description**
> Verify the login flow works end-to-end.
>
> **## Priority**
> `blocking`
>
> **## Setup**
> ```bash
> npm start &
> sleep 3
> ```
>
> **## Steps**
> 1. Open browser to http://localhost:3000
> 2. Click the "Login" button
> 3. Enter credentials
> 4. Verify the dashboard loads
>
> **## Expected Output**
> Dashboard shows "Welcome" text. No console errors.

- **Priority**: `blocking` = merge-gating, `warning` = advisory
- **Commands**: Real shell commands, executed inside Docker
- **Steps**: Browser actions, executed by IDE via Chrome DevTools

## Architecture

revv has two components:

### Skills (the brain)

| Skill | What it does |
|-------|-------------|
| `revv-update` | Reads codebase + changes → generates/updates `.revv/` tests |
| `revv-run` | Runs update → executes Docker tests → runs browser tests → analyzes failures |

Skills live in [`skills/`](skills/) and are fetched by the IDE from this repo via `AGENTS.md`.

### Binary (optional accelerator)

```
revv exec [flags]
  --verbose     Show detailed output
  --json        Output results as JSON
  --category    Run only a specific category
  --test        Run a single test by path
  --timeout     Global timeout (default: 5m)
```

The binary runs tests in parallel Docker containers. It's optional — the skill can build it from source (`go build ./cmd/revv`) or fall back to raw `docker run` commands.

```
cmd/revv/          ← binary entrypoint
internal/
├── cli/           ← cobra commands (exec, version)
├── runner/        ← test discovery, parsing, parallel execution
├── sandbox/       ← Docker container lifecycle (Colima auto-install)
└── git/           ← git utilities
```

## FAQ

**Do contributors need an API key?**
No. The IDE's built-in LLM handles everything.

**Do contributors need to install anything?**
No. They just clone the repo and say "revv run". Docker (via Colima) is auto-installed if needed.

**Does AGENTS.md go stale?**
No. It's a pointer that fetches the latest instructions from this repo every time.

**What IDEs are supported?**
Any IDE with an LLM that reads `AGENTS.md` — Antigravity, Codex, Claude Code, Cursor.

**What about CI?**
Use `revv exec` (Go binary) or raw Docker commands. No LLM needed in CI.

## License

MIT
