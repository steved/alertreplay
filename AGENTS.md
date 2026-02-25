# Repository Guidelines

## Build, Test, and Development Commands
- **Build**: `go build`
- **Test**: `mise r test` or `mise r test ./path/to/package`
- **Lint**: `mise r lint` or `mise r lint ./path/to/package`
- **Type checking**: `go vet`
- **Format**: `mise r fmt`

Tooling: Golang `1.26.0`. Non-Go tooling for CI/k8s workflows is pinned in `mise.toml` (`mise install`).

## Coding Style & Naming Conventions
- Always write tests for new functional code
- If modifying existing code that is not tested, write new tests
- When in doubt, or in the presence of multiple options, ask the user
- Prompt for confirmation before making changes unless otherwise directed

## Testing Guidelines
- Before committing code:
  - Code must be formatted using `mise r fmt`
  - All tests must pass (any new or modified tests)
  - Linting must not return any issues

## Commit & Pull Request Guidelines
- The repo follows [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/); prefixes like `feat: `, `fix: `, `chore: `, etc are required
- Keep PRs small and include: purpose, how you tested (`mise r test`, config used).
