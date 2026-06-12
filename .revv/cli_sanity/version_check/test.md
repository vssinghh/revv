## Description
Verify that the version command prints correct metadata and exits successfully.

## Priority
blocking

## Commands
```bash
./bin/revv version 2>&1 | grep -q "revv" || (echo "FAIL: missing 'revv' prefix" && exit 1)
./bin/revv version 2>&1 | grep -q "commit:" || (echo "FAIL: missing 'commit:' info" && exit 1)
./bin/revv version 2>&1 | grep -q "built at:" || (echo "FAIL: missing 'built at:' info" && exit 1)
echo "PASS: version output is correct"
```

## Expected Output
Exit code 0. Output ends with "PASS: version output is correct".
