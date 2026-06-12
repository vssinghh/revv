## Description
Verify that running 'revv init' without GEMINI_API_KEY unsets returns an actionable error.

## Priority
blocking

## Commands
```bash
unset GEMINI_API_KEY
./bin/revv init 2>&1 | grep -q "GEMINI_API_KEY environment variable is not set" || (echo "FAIL: error message missing" && exit 1)
echo "PASS: missing API key handled successfully"
```

## Expected Output
Exit code 0. Output ends with "PASS: missing API key handled successfully".
