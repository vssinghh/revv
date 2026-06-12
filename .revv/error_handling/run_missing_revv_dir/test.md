## Description
Verify that 'revv run' errors out gracefully when .revv configuration directory is missing.

## Priority
blocking

## Commands
```bash
mv .revv .revv_backup
./bin/revv run 2>&1 | grep -q "no .revv/ directory found"
STATUS=$?
mv .revv_backup .revv
if [ $STATUS -ne 0 ]; then
  echo "FAIL: run did not complain about missing .revv"
  exit 1
fi
echo "PASS: missing directory handled successfully"
```

## Expected Output
Exit code 0. Output ends with "PASS: missing directory handled successfully".
