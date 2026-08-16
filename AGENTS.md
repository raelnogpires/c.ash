# Repository Guidelines

## Project Structure & Module Organization

`c.ash` is an early-stage desktop personal-finance application planned in Go. The repository currently contains only [`README.md`](README.md), which describes the product goals, cross-platform target, local data storage, and visual themes. As implementation begins, keep production code in `cmd/` and `internal/`, tests beside the packages they exercise, and static UI assets in `assets/`. Record substantial architectural choices in the README or a focused design note.

## Build, Test, and Development Commands

No build tooling or Go module exists yet. Once the project is initialized, use the standard Go workflow from the repository root:

- `go run ./cmd/cash` — run the desktop application locally.
- `go build ./...` — compile every package and catch integration errors.
- `go test ./...` — run all unit and integration tests.
- `go vet ./...` — detect common correctness issues.
- `gofmt -w <files>` — format Go source before committing.

Update this section when the chosen GUI toolkit, packaging scripts, or platform-specific commands are added.

## Coding Style & Naming Conventions

Follow idiomatic Go and let `gofmt` define formatting: tabs for indentation, short package names, exported identifiers documented with Go-style comments, and `MixedCaps` for identifiers. Use `snake_case` only where required by external files or data formats. Keep business logic independent from UI code, validate user input at boundaries, and make local-storage access explicit and testable.

## Testing Guidelines

Use Go’s standard `testing` package unless the project adopts a documented alternative. Name files `*_test.go` and tests `Test<Subject>_<Scenario>` (for example, `TestTransactionStore_Add`). Cover calculations, persistence, validation, and theme-independent behavior; do not rely on an interactive desktop session for unit tests. Run `go test ./...` and `go vet ./...` before submitting changes.

## Commit & Pull Request Guidelines

There is no Git commit history yet, so no repository-specific convention is established. Use concise imperative messages with a small scope, such as `add transaction persistence` or `fix monthly total calculation`. Pull requests should explain the user-facing change, identify tests run, link a relevant issue when one exists, and include screenshots or a short recording for UI changes. Keep unrelated refactors out of feature changes.

## Security & Configuration

Keep personal financial data local by default and never commit real user data, secrets, credentials, or machine-specific configuration. Use sample or fixture data in tests, and document any migration or storage-format change before merging.
