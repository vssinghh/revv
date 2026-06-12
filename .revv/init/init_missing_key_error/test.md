## Description
Verify that 'revv init' aborts gracefully with a specific error message and exit code 1 when GEMINI_API_KEY is not defined in the environment.

## Priority
blocking

## Commands
```bash
# Unset key and attempt to run init
unset GEMINI_API_KEY

output=$(./bin/revv init 2>&1)
exit_code=$?

if [ $exit_code -ne 1 ]; then
  echo "FAIL: Expected exit code 1 when GEMINI_API_KEY is unset, got $exit_code"
  exit 1
fi

if ! echo "$output" | grep -q "GEMINI_API_KEY environment variable is not set"; then
  echo "FAIL: Incorrect error message output: $output"
  exit 1
fi

echo "PASS: Unset API key is correctly validated and throws specific error"
```

## Expected Output
Exit code 0. Final line: "PASS: Unset API key is correctly validated and throws specific error".
