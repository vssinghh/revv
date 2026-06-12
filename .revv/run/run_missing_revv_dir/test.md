## Description
Verify that the 'revv run' command fails gracefully with exit code 1 and a descriptive error message when executed in a directory that has not been initialized with .revv.

## Priority
blocking

## Commands
```bash
tmpdir="/tmp/test_run_norun"
rm -rf "$tmpdir"
mkdir -p "$tmpdir"
cd "$tmpdir"

output=$(/workspace/bin/revv run 2>&1)
exit_code=$?

if [ $exit_code -ne 1 ]; then
  echo "FAIL: Expected exit code 1 for missing .revv dir, got $exit_code"
  exit 1
fi

if ! echo "$output" | grep -q "no .revv/ directory found"; then
  echo "FAIL: Unexpected error output: $output"
  exit 1
fi

echo "PASS: Execution with missing review configurations fails gracefully"
```

## Expected Output
Exit code 0. Final line: "PASS: Execution with missing review configurations fails gracefully".
