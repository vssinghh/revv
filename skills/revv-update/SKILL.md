---
name: revv-update
description: Generate or update automated QA tests for a code repository. Reads the codebase, analyzes local changes, and creates/updates the .revv/ directory with Dockerfile, test definitions, and helper scripts. Use when the user wants to set up revv for the first time, update tests after code changes, or generate new test coverage. Trigger on 'revv update', 'update tests', 'generate tests', 'create revv config', 'set up revv', 'add tests'.
---

## What is revv?

revv is an IDE-first QA automation convention. Tests are defined as markdown files in a `.revv/` directory and executed inside Docker containers. No binary or API key required — the IDE's LLM handles all intelligence.

## What to do

### 1. Read the codebase context

- Project structure (ls the root directory)
- Build system: `Makefile`, `package.json`, `Cargo.toml`, etc.
- Existing docs: `README.md`, `CONTRIBUTING.md`
- Source files relevant to recent changes

### 2. Read local changes

- Run `git diff HEAD` to see uncommitted changes
- Run `git log --oneline -5` for recent commits
- If on a branch, run `git diff main...HEAD` for the full PR diff

### 3. Read existing `.revv/` directory (if any)

- `.revv/Dockerfile` — current sandbox configuration
- All `.revv/<category>/<test_name>/test.md` files — existing tests
- `.revv/helpers/` — shared helper scripts

### 4. Generate or update the `.revv/` directory

**Directory structure:**
```
.revv/
├── Dockerfile
├── helpers/          ← shared helper scripts
│   └── assert.sh
├── build/
│   └── compile_check/
│       └── test.md
├── unit/
│   └── json_parsing/
│       └── test.md
└── manual/
    └── login_flow/
        └── test.md
```

**For each automated test, create a `test.md` with this exact format:**
```markdown
## Description
What this test verifies and why it matters.

## Priority
blocking | warning

## Commands
```bash
# Real, executable shell commands. NOT pseudocode.
make build
test -x ./bin/myapp || (echo "FAIL: binary not found" && exit 1)
echo "PASS"
```

## Expected Output
Exit code 0. Describe what success looks like.
```

**For manual/visual/browser tests:**
```markdown
## Description
Verify the login flow works end-to-end.

## Priority
blocking

## Setup
```bash
# Commands to start the app (run in Docker or locally)
npm start &
sleep 3
```

## Steps
1. Open browser to http://localhost:3000
2. Click the "Login" button
3. Enter "test@example.com" in the email field
4. Enter "password123" in the password field
5. Click "Submit"
6. Verify the page shows "Welcome, Test User"
7. Take a screenshot of the dashboard

## Expected Output
The dashboard loads with "Welcome, Test User" visible.
No console errors. All elements render correctly.
```

> **Note:** Manual tests with `## Steps` are executed by the IDE's
> browser automation (see the "revv-run" skill). They are NOT run
> inside Docker — the binary skips tests without `## Commands`.

**`.revv/Dockerfile` rules:**
- Base image appropriate for the project (e.g., `golang:1.22-alpine`, `node:20-alpine`)
- Install all build dependencies
- COPY source code into `/workspace`
- **MUST pre-build the project** (e.g., `RUN make build`) so tests start instantly
- Working directory: `/workspace`

### 5. Present changes to the developer for review before committing.

## Rules

- Every test.md MUST have: `## Description`, `## Priority`, `## Commands` (or `## Steps` for manual), `## Expected Output`
- Commands must be REAL shell commands that work inside the Docker container
- Priority is either `blocking` (merge-gating) or `warning` (advisory)
- Do NOT use mock modes, test doubles, or fake backends
- Do NOT generate tests for things already covered by existing `.revv/` tests
- When updating, preserve tests that are still relevant. Only add/modify/delete as needed
- Always include a `manual/` category for tests that cannot be automated (UI, UX, visual)
- If this is the first time (no `.revv/` exists), also create an `AGENTS.md` in the repo root with this exact content:

```markdown
# revv — Automated QA

This project uses [revv](https://github.com/vssinghh/revv) for automated QA testing.
Tests are defined in the `.revv/` directory and run inside Docker containers.

## Available commands

When you say "revv update" or "revv run", read the skill instructions from:

- **revv update**: https://raw.githubusercontent.com/vssinghh/revv/main/skills/revv-update/SKILL.md
- **revv run**: https://raw.githubusercontent.com/vssinghh/revv/main/skills/revv-run/SKILL.md

Read the instructions from the URLs above and follow them exactly.
```

