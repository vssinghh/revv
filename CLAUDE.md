# revv — Automated QA for Open Source

revv is an IDE-first QA tool. You (the IDE's LLM) handle test generation and analysis. The `revv exec` binary handles Docker execution.

## Skill: revv update

### When to trigger
User says: "revv update", "update tests", "generate tests", "create revv config"

### What to do

1. **Read the codebase context**:
   - Project structure (ls the root directory)
   - Build system: `Makefile`, `package.json`, `Cargo.toml`, etc.
   - Existing docs: `README.md`, `CONTRIBUTING.md`
   - Source files relevant to recent changes

2. **Read local changes**:
   - Run `git diff HEAD` to see uncommitted changes
   - Run `git log --oneline -5` for recent commits
   - If on a branch, run `git diff main...HEAD` for the full PR diff

3. **Read existing `.revv/` directory** (if any):
   - `.revv/Dockerfile` — current sandbox configuration
   - All `.revv/<category>/<test_name>/test.md` files — existing tests
   - `.revv/helpers/` — shared helper scripts

4. **Generate or update the `.revv/` directory**:

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
       └── dark_mode/
           └── test.md
   ```

   **For each test, create a `test.md` with this exact format:**
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

   **For manual/visual tests (no automation possible):**
   ```markdown
   ## Description
   Visual check that dark mode renders correctly.

   ## Priority
   warning

   ## Steps
   1. Open the app in a browser
   2. Toggle dark mode
   3. Verify all text is readable

   ## Expected Output
   All UI elements should be visible and properly themed.
   ```

   **`.revv/Dockerfile` rules:**
   - Base image appropriate for the project (e.g., `golang:1.22-alpine`, `node:20-alpine`)
   - Install all build dependencies
   - COPY source code into `/workspace`
   - **MUST pre-build the project** (e.g., `RUN make build`) so tests start instantly
   - Working directory: `/workspace`

5. **Present changes to the developer** for review before committing.

### Rules
- Every test.md MUST have: `## Description`, `## Priority`, `## Commands` (or `## Steps` for manual), `## Expected Output`
- Commands must be REAL shell commands that work inside the Docker container
- Priority is either `blocking` (merge-gating) or `warning` (advisory)
- Do NOT use mock modes, test doubles, or fake backends
- Do NOT generate tests for things already covered by existing `.revv/` tests
- When updating, preserve tests that are still relevant. Only add/modify/delete as needed.
- Always include a `manual/` category for tests that cannot be automated

---

## Skill: revv run

### When to trigger
User says: "revv run", "run tests", "test this", "run revv"

### What to do

1. **First, run the "revv update" skill** to check if the current changes need new or updated tests. If `.revv/` doesn't exist at all, generate it from scratch.

2. **Execute tests** by running:
   ```bash
   revv exec --verbose
   ```
   Or with filters:
   ```bash
   revv exec --category unit --verbose
   revv exec --test build/compile_check --verbose
   ```
   Or for machine-readable output:
   ```bash
   revv exec --json
   ```

3. **Read the output** and summarize results to the developer.

4. **If any tests failed**, analyze the failure:
   - Read the test.md that failed
   - Read the error output
   - Read the relevant source code
   - Explain WHY it failed (test bug vs code bug vs environment issue)
   - Suggest a specific fix

5. **Report coverage gaps**: If the current changes introduce code that isn't covered by any test, tell the developer.

### Example interaction
```
User: revv run

A: I'll check if your changes need new tests, then run the suite.

[Updates .revv/ with 1 new test for the added --timeout flag]

Running tests...
$ revv exec --verbose

  ✓ build/compile_check        blocking  PASS  (0.5s)
  ✓ cli_sanity/version_check   blocking  PASS  (0.2s)
  ✗ cli_sanity/timeout_flag    blocking  FAIL  (0.1s)

Results: 2 passed, 1 failed

❌ cli_sanity/timeout_flag failed because the --timeout flag
expects a duration string like "5m" but the test passes "300"
(an integer). Fix: change the test command from
`./bin/app --timeout 300` to `./bin/app --timeout 5m`.
```

---

## revv exec Reference

The `revv exec` binary is the Docker test executor. It has NO LLM logic.

```
revv exec [flags]
  --category string   Run only a specific category
  --test string       Run a single test by path
  --verbose           Show detailed output
  --json              Output results as JSON
  --timeout duration  Global timeout (default: 5m)
```

JSON output format:
```json
{
  "results": [
    {"category": "build", "name": "compile_check", "priority": "blocking", "passed": true, "duration": 0.5},
    {"category": "unit", "name": "parser_test", "priority": "warning", "passed": false, "error": "exit code 1", "output": "..."}
  ],
  "summary": {"passed": 5, "failed": 1, "skipped": 0, "blocking_passed": 4, "blocking_total": 5}
}
```
