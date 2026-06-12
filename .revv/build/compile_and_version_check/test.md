## Description
Verify that the CLI binary compiles cleanly using the Makefile and check that the version subcommand returns the correct formatted properties.

## Priority
blocking

## Commands
```bash
# Build the binary
make build

# Check binary exists and is executable
test -x ./bin/revv || (echo "FAIL: revv binary not found or not executable" && exit 1)

# Run version subcommand and assert correct naming
./bin/revv version | grep -q "revv" || (echo "FAIL: version subcommand output invalid" && exit 1)

echo "PASS: CLI binary builds and outputs correct version signature"
```

## Expected Output
Exit code 0. Output contains "PASS: CLI binary builds and outputs correct version signature".
