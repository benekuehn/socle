# Repository Guidelines

## Project Structure & Module Organization
- Run CLI work from `so/`; `main.go` wires the Cobra root command to `cmd/`.
- Commands live in `cmd/` as `<verb>.go`, `<verb>_runner.go`, `<verb>_test.go`.
- Shared utilities sit under `internal/` (notably `exec`, `gh`, `git`, `ui`, `docgen`, `testutils`).
- Build artifacts go to `../bin/`; CLI reference in `README.md` is generated via `internal/docgen`.

## Build, Test, and Development Commands
- `make build` — compile the `so` binary to `../bin/so`.
- `make dev` — install `so-dev` to `$GOPATH/bin` for iterative testing.
- `make test` / `go test ./...` — run the full Go test suite.
- `make fmt` — apply `gofmt` to every package.
- `make lint` / `make lint-fix` — run `golangci-lint` (auto-fix with `--fix`).
- `make docs` — regenerate CLI docs into `README.md`; `make all` runs fmt + lint + test.

## Coding Style & Naming Conventions
- Standard Go style: tabs for indentation; `UpperCamelCase` for exports, `lowerCamelCase` for internals.
- Group command files per verb and keep functions small and focused.
- Rely on `gofmt` and `golangci-lint`; avoid suppressing lint warnings.

## Testing Guidelines
- Use the Go `testing` package; place tests beside implementations.
- Name tests `Test<Command>` or `Test<PackageScenario>` to mirror the target.
- For git-heavy scenarios, reuse helpers in `internal/testutils` (e.g., `SetupGitRepo`) instead of shelling out.
- Run `go test -cover ./...` before opening a PR; add focused tests for new behaviors.

## Commit & Pull Request Guidelines
- Follow Conventional Commits with optional scopes (e.g., `feat: cmd/up`, `fix: cmd/create`).
- Keep commits focused and include necessary test updates.
- PRs should state user impact, list manual verification steps, and link Jira/GitHub issues.
- Attach screenshots or terminal captures for CLI output changes; note any follow-up tasks explicitly.
