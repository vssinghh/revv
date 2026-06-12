# AGENTS.md — Revv

Revv is an LLM-powered PR review automation tool for open-source maintainers. It generates test configurations and executes them in isolated Docker sandboxes.

## Quick Reference

```bash
# Build
make build

# Run all tests
go test ./...

# Generate .revv/ config for a repo (needs GEMINI_API_KEY)
GEMINI_API_KEY=<key> ./bin/revv init

# Execute generated tests in Docker sandbox
GEMINI_API_KEY=<key> ./bin/revv run --verbose
```

## Architecture

```
cmd/revv/main.go          → CLI entrypoint
internal/cli/              → Cobra commands (init, run, version)
internal/adk/client.go     → LLM prompt construction + Gemini API call
internal/llm/llm.go        → JSON schema, response parsing, mock mode
internal/runner/            → Test discovery, parsing, parallel execution
internal/sandbox/           → Docker container lifecycle, socket discovery, auto-install
internal/context/           → Repository file collection for LLM context
internal/git/               → Branch creation, staging, committing
```

## Key Design Decisions

- **One container per test**: Each test gets a fresh Docker container from a cached image. No shared state between tests.
- **Parallel execution**: All tests run concurrently via goroutines. Results are collected via indexed slice (no channel needed).
- **Env var auto-detection**: The runner scans `test.md` files for `$VARIABLE` references and auto-forwards matching host env vars into containers.
- **Priority normalization**: The parser maps LLM-generated priorities (high/critical/p0 → blocking, low/medium → warning) since LLMs don't always use exact terms.
- **Docker socket discovery**: Supports Colima (`~/.colima/`), Rancher Desktop (`~/.rd/`), and standard Docker (`/var/run/docker.sock`).
- **Mock LLM**: Set `REVV_MOCK_LLM=true` for unit/E2E tests. Never use mocks in generated `.revv/` tests — those must exercise real product behavior.

## Making Changes

### Adding a new CLI command
1. Create `internal/cli/<command>.go` with a `newXxxCmd()` function
2. Register it in `internal/cli/root.go` via `rootCmd.AddCommand()`
3. Add E2E tests in `tests/e2e/e2e_test.go`

### Modifying the LLM prompt
1. Edit `internal/adk/client.go` → `BuildPrompt()` function
2. The prompt is constructed programmatically (not a template file)
3. Test by running `revv init` with a real API key and checking the generated output

### Modifying test parsing
1. Edit `internal/runner/parser.go`
2. Update `internal/runner/parser_test.go` with new cases
3. Priority values must go through `normalizePriority()`

### Modifying the sandbox
1. Edit `internal/sandbox/sandbox.go`
2. `Exec()` creates a fresh container per call — keep it thread-safe (uses mutex for container tracking)
3. `ensureDockerHost()` handles socket discovery — add new paths there for other runtimes

## Validation Checklist

Before submitting a PR:

1. `go build ./cmd/revv` — must compile
2. `go test ./...` — all unit + E2E tests must pass
3. `GEMINI_API_KEY=<key> ./bin/revv init` — must generate valid `.revv/` config
4. `./bin/revv run --verbose` — generated tests must pass in Docker sandbox

## Common Pitfalls

- **Don't use `<angle brackets>` in Go string literals** in `client.go` — the Go compiler parses them as generics syntax
- **`revv init` creates a `revv/init` branch** — switch back to `main` before committing source changes
- **`.revv/` is not gitignored** — it's meant to be committed on the `revv/init` branch, not on `main`
- **The mock LLM (`REVV_MOCK_LLM`)** uses priorities like "High"/"Low" — the parser normalizes these, but keep the mock data realistic
