# Repository Guidelines

## Project Structure & Module Organization
- Run CLI development from `so/`; `main.go` wires the Cobra root command and delegates to `cmd/`.
- `so/cmd/` holds top-level commands; each command has a handler (`<name>.go`), runner (`<name>_runner.go`), and test (`<name>_test.go`).
- Shared utilities live under `so/internal/` (notably `exec` for subprocesses, `gh`/`git` integrations, `ui` for prompts, `docgen` for README generation, and `testutils` for git fixture helpers).
- Generated or installed artifacts reside outside the module: builds land in `bin/`, developer docs in `so/README.md`.

## Build, Test, and Development Commands
- `make build` (from `so/`) compiles the `so` binary into `../bin/so`.
- `make dev` installs a `so-dev` binary to your `$GOPATH/bin` for iterative testing.
- `make test` or `go test ./...` executes the full Go test suite.
- `make fmt` applies `gofmt` to every package; run before committing to avoid diffs.
- `make lint` requires `golangci-lint`; `make lint-fix` attempts auto-fixes.

## Coding Style & Naming Conventions
- Follow standard Go style: tabs for indentation, `UpperCamelCase` for exported symbols, `lowerCamelCase` for internals.
- Keep command files grouped (`<verb>.go`, `<verb>_runner.go`, `<verb>_test.go`) and prefer small, focused functions.
- Rely on `go fmt` and `golangci-lint` to enforce formatting, imports, and vetting; resolve all warnings rather than suppressing them.

## Testing Guidelines
- Use the standard `testing` package; co-locate tests beside implementation.
- Name tests `Test<Command>` or `Test<PackageScenario>`, mirroring the command or helper under test.
- For git-dependent scenarios, reuse helpers in `so/internal/testutils` (e.g., `SetupGitRepo`) instead of shelling out manually.
- Run `go test -cover ./...` before opening a PR when changes touch command logic.

## Commit & Pull Request Guidelines
- Follow Conventional Commits (`feat:`, `fix:`, `chore:`) as seen in `git log`; scope commands with `/` when helpful (e.g., `feat: cmd/up`).
- Keep commits focused and include relevant test updates.
- Pull requests should describe user impact, list manual verification steps, and link Jira or GitHub issues where applicable.
- Attach screenshots or terminal captures when behavior changes the CLI output; note any follow-up tasks explicitly.
