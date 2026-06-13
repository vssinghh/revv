---
name: revv-run
description: Execute all revv QA tests for the current repository. First runs revv-update to check if new tests are needed, then executes automated tests via the revv binary and manual tests via browser automation. Analyzes failures and reports coverage gaps. Trigger on 'revv run', 'run tests', 'test this', 'run revv', 'test my changes'.
---

## What to do

### 1. Run revv-update first

Check if the current changes need new or updated tests. If `.revv/` doesn't exist at all, generate it from scratch using the `revv-update` skill.

### 2. Execute automated tests (tests with `## Commands`)

These are deterministic shell commands — no LLM needed. Use the `revv exec` binary for fast, parallel execution.

**Ensure the binary is available:**
```bash
which revv || go build -o /tmp/revv ./cmd/revv
```

**Run all automated tests:**
```bash
revv exec --verbose
# or if built to /tmp:
/tmp/revv exec --verbose
```

The binary:
- Builds the Docker image from `.revv/Dockerfile` (once, cached)
- Runs each test in a fresh container (parallel, worker pool)
- Collects results with exit codes and output
- Supports `--json` for structured output

### 3. Execute browser tests (tests with `## Steps`)

These require intelligence — the binary skips them. You handle them directly.

For each test that has `## Steps` instead of `## Commands`:

a. **Run `## Setup`** commands first (if present) — these start the app, seed the database, etc.

b. **Open a browser** using Chrome DevTools MCP tools:
   - Use `new_page` or `navigate_page` to open the target URL
   - Use `click`, `fill`, `type` to interact with elements
   - Use `get_text`, `evaluate_javascript` to verify content
   - Use `screenshot` to capture visual state

c. **Follow each step literally:**
   - "Open browser to http://localhost:3000" -> navigate_page
   - "Click the Login button" -> find the button, click
   - "Enter email" -> fill the input field
   - "Verify page shows X" -> get_text and check for X
   - "Take a screenshot" -> screenshot and embed in results

d. **Report results** with screenshots for visual verification.

e. **If browser tools are not available**, print the steps for the developer to execute manually and mark the test as "manual - needs human verification".

### 4. Summarize results

Combine results from both the binary (automated) and browser tests:
- Which tests passed/failed
- Total duration
- Blocking vs warning status

### 5. Analyze failures

If any tests failed:
- Read the test.md that failed
- Read the error output / screenshot
- Read the relevant source code
- Explain WHY it failed (test bug vs code bug vs environment issue)
- Suggest a specific fix

### 6. Report coverage gaps

If the current changes introduce code that isn't covered by any test, tell the developer.

## Examples

### Automated test flow

User says "revv run". The assistant:
1. Runs revv-update (finds 1 new test needed for a --timeout flag change)
2. Builds the binary: `go build -o /tmp/revv ./cmd/revv`
3. Runs `/tmp/revv exec --verbose`
4. Gets: 2 passed, 1 failed (parallel, fast)
5. Analyzes the failure: "timeout_flag failed because --timeout expects '5m' not '300'"
6. Suggests the fix

### Browser test flow

User says "revv run". The assistant:
1. Runs automated tests via binary (all pass)
2. Finds manual/login_flow test with `## Steps`
3. Runs `## Setup` to start the app
4. Opens browser, navigates to localhost:3000
5. Clicks Login, fills credentials, submits
6. Verifies "Welcome" text appears
7. Takes screenshot, reports pass
