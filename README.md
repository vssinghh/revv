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

Open your IDE (Antigravity, Claude Code, Cursor, Codex) and paste this prompt:

> Read https://raw.githubusercontent.com/vssinghh/revv/main/skills/revv-update/SKILL.md and follow the instructions to set up automated QA for this repo.

Your IDE will:
1. Fetch the latest revv instructions
2. Analyze your codebase
3. Generate `.revv/` with Dockerfile and test definitions
4. Generate `AGENTS.md` so contributors can use revv too

Review the generated files, commit, and push. That's it.

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
> **## Type**
> `automated`
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
> **## Type**
> `browser`
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

```
IDE Skill (the brain)
│
├── "revv update"  → LLM generates/updates .revv/ tests
│
├── "revv run"
│   ├── Automated tests → binary (fast, parallel Docker containers)
│   ├── Browser tests   → IDE directly (Chrome DevTools MCP)
│   └── Analyze results → LLM explains failures, suggests fixes
```

| Component | What | Why |
|-----------|------|-----|
| [`revv-update`](skills/revv-update/SKILL.md) | Generates `.revv/` tests | Needs LLM to understand code |
| [`revv-run`](skills/revv-run/SKILL.md) | Orchestrates test execution | Needs LLM for browser tests + analysis |
| `revv exec` (Go binary) | Parallel Docker test runner | Fast, no LLM needed, self-builds from source |

Skills are fetched by the IDE from this repo via the `AGENTS.md` pointer. The binary is built from source automatically — no install.

## FAQ

**Do contributors need an API key?**
No. The IDE's built-in LLM handles everything.

**Do contributors need to install anything?**
No. They just clone the repo and say "revv run". Docker (via Colima) is auto-installed if needed.

**Does AGENTS.md go stale?**
No. It's a pointer that fetches the latest instructions from this repo every time.

**What IDEs are supported?**
Any IDE with an LLM that reads `AGENTS.md` — Antigravity, Codex, Claude Code, Cursor.

## License

MIT
