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
   > browser automation (see "revv run" skill below). They are NOT
   > run by `revv exec` — the binary skips tests without `## Commands`.

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

2. **Execute automated tests** (tests with `## Commands`):

   **If `revv exec` is installed** (preferred — parallel, fast):
   ```bash
   revv exec --verbose
   ```
   Or with filters:
   ```bash
   revv exec --category unit --verbose
   ```

   **If `revv exec` is NOT installed** (fallback — sequential):
   ```bash
   # Build the sandbox image once
   docker build -t revv-sandbox -f .revv/Dockerfile .

   # Run each test in its own container
   docker run --rm revv-sandbox sh -c "<commands from test.md>"
   ```
   Discover tests by finding all `.revv/<category>/<name>/test.md` files,
   extract the `## Commands` code block, and run each one.

3. **Execute manual/browser tests** (tests with `## Steps`):

   For each test that has `## Steps` instead of `## Commands`:

   a. **Run `## Setup`** commands first (if present) — these start the app, seed the database, etc.

   b. **Open a browser** using Chrome DevTools MCP tools:
      - Use `new_page` or `navigate_page` to open the target URL
      - Use `click`, `fill`, `type` to interact with elements
      - Use `get_text`, `evaluate_javascript` to verify content
      - Use `screenshot` to capture visual state

   c. **Follow each step literally:**
      - "Open browser to http://localhost:3000" → `navigate_page({url: "http://localhost:3000"})`
      - "Click the Login button" → find the button, `click({selector: "button:has-text('Login')"})`
      - "Enter email" → `fill({selector: "input[type=email]", value: "test@example.com"})`
      - "Verify page shows X" → `get_text({selector: "body"})` and check for X
      - "Take a screenshot" → `screenshot()` and embed in the results

   d. **Report results** with screenshots for visual verification.

   e. **If browser tools are not available**, print the steps for the developer to execute manually and mark the test as "manual — needs human verification".

4. **Read the output** and summarize results to the developer.

5. **If any tests failed**, analyze the failure:
   - Read the test.md that failed
   - Read the error output / screenshot
   - Read the relevant source code
   - Explain WHY it failed (test bug vs code bug vs environment issue)
   - Suggest a specific fix

6. **Report coverage gaps**: If the current changes introduce code that isn't covered by any test, tell the developer.

### Example: automated test
```
User: revv run

Assistant: I'll check if your changes need new tests, then run the suite.

[Updates .revv/ with 1 new test for the added --timeout flag]

Running automated tests...
$ revv exec --verbose

  ✓ build/compile_check        blocking  PASS  (0.5s)
  ✓ cli_sanity/version_check   blocking  PASS  (0.2s)
  ✗ cli_sanity/timeout_flag    blocking  FAIL  (0.1s)

Results: 2 passed, 1 failed

❌ cli_sanity/timeout_flag failed because the --timeout flag
expects a duration string like "5m" but the test passes "300".
Fix: change `./bin/app --timeout 300` to `./bin/app --timeout 5m`.
```

### Example: browser test
```
User: revv run

Assistant: Running automated tests...
$ revv exec --verbose
  ✓ build/compile_check   blocking  PASS  (0.5s)

Running browser tests...
  🌐 manual/login_flow:
    ✓ Opened http://localhost:3000
    ✓ Clicked "Login" button
    ✓ Entered credentials
    ✓ Clicked "Submit"
    ✓ Dashboard shows "Welcome, Test User"
    📸 Screenshot captured

  🌐 manual/dark_mode:
    ✓ Opened settings page
    ✓ Toggled dark mode
    ✗ FAIL: contrast ratio on sidebar text is 2.1:1 (minimum 4.5:1)
    📸 Screenshot captured

All automated tests passed. 1 browser test failed:
❌ manual/dark_mode: sidebar text has insufficient contrast in dark mode.
Fix: change --sidebar-text from #666 to #aaa in theme.css.
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
