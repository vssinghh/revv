## Description
Verify that the CLI help message displays standard layout, formatting, and subcommands.

## Priority
warning

## Commands
```bash
./bin/revv --help 2>&1 | grep -q "LLM-powered PR review automation tool" || (echo "FAIL: root help layout incorrect" && exit 1)
./bin/revv init --help 2>&1 | grep -q "Initialize revv" || (echo "FAIL: init help layout incorrect" && exit 1)
./bin/revv run --help 2>&1 | grep -q "Run tests from .revv/" || (echo "FAIL: run help layout incorrect" && exit 1)
echo "PASS: visual layout check successful"
```

## Expected Output
Exit code 0. Output ends with "PASS: visual layout check successful".
