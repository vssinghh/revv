---
name: revv-run
description: Execute all revv QA tests for the current repository. First runs revv-update to check if new tests are needed, then executes automated tests in Docker containers and manual tests via browser automation. Analyzes failures and reports coverage gaps. Trigger on 'revv run', 'run tests', 'test this', 'run revv', 'test my changes'.
---

## What to do

### 1. Run revv-update first

Check if the current changes need new or updated tests. If `.revv/` doesn't exist at all, generate it from scratch using the `revv-update` skill.

### 2. Execute automated tests (tests with `## Commands`)

Find all `.revv/<category>/<name>/test.md` files that have a `## Commands` section.

**Step A: Ensure `revv exec` is available**
```bash
# Check if already available
which revv

# If not found, build from source (the code is in this repo)
go build -o /tmp/revv ./cmd/revv
# Then use /tmp/revv exec --verbose
```

**Step B: Run the tests**
```bash
revv exec --verbose
```

**If Go is also not installed** (final fallback — raw docker):
```bash
# Build the sandbox image once
docker build -t revv-sandbox -f .revv/Dockerfile .

# For each test.md with ## Commands, run in its own container:
docker run --rm revv-sandbox sh -c "<commands from test.md>"
```

### 3. Execute manual/browser tests (tests with `## Steps`)

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

Report all results to the developer:
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
2. Runs `revv exec --verbose`
3. Gets: 2 passed, 1 failed
4. Analyzes the failure: "cli_sanity/timeout_flag failed because --timeout expects '5m' not '300'"
5. Suggests the fix

### Browser test flow

User says "revv run". The assistant:
1. Runs automated tests via `revv exec` (all pass)
2. Finds manual/login_flow test with `## Steps`
3. Runs `## Setup` to start the app
4. Opens browser, navigates to localhost:3000
5. Clicks Login, fills credentials, submits
6. Verifies "Welcome" text appears
7. Takes screenshot, reports pass

