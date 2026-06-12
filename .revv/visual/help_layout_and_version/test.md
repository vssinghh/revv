## Description
Inspect CLI help commands and make sure subcommands are correctly presented with balanced formatting and clean alignments.

## Priority
warning

## Commands
```bash
echo "=== Global Help Layout ==="
./bin/revv --help
echo ""
echo "=== Init Help Layout ==="
./bin/revv init --help
echo ""
echo "=== Run Help Layout ==="
./bin/revv run --help
```

## Expected Output
No validation assertions. Reviewer checks CLI layout manually to ensure flag options, spacing, alignment, and command descriptions look professional.
