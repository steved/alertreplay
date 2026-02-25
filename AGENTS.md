# Agent Guidelines

## Commands
- **Code search**: Use `ast-grep` (ast-grep) when you need to match and retrieve complete syntactic structures (functions, classes, methods, expressions, etc.). A good example for finding a receiver function named RegisterRoutes in go `ast-grep --lang go --pattern 'func ($_) RegisterRoutes($$$)'`
- **Build**: `go build`
- **Test**: `mise r test` or `mise r test ./path/to/package`
- **Lint**: `mise r lint` or `mise r lint ./path/to/package`
- **Sanity checking (tests + lint)**: `mise r test:sanity` NOTE: This command runs the entire test suite and linter concurrently, which typically takes 3-5 minutes. To avoid timeouts, prefer running tests on specific packages (e.g., `mise r test ./api` then `mise r lint ./api`) instead of the full sanity check. If you must run the full suite, ensure your tool supports extended execution times.
- **Type checking**: `go vet`
- **Format**: `mise r fmt`
- **Coverage reporting**: `mise r cover` or `mise r cover ./path/to/package`
- **CI check**: Run all tests, then lint; fail on any failed tests or lint diagnostics
- **GitHub interaction**: Use the `gh` CLI tool to interact with GitHub, for example to create pull requests

## Workflow
- **Coding**:
  - When writing code, always adhere to the guidelines in this document
  - Always write tests for new functional code
  - If modifying existing code that is not tested, write new tests
  - When in doubt, or in the presence of multiple options, ask the user
  - Prompt for confirmation before making changes unless otherwise directed
  - Prefer running specific test packages (`mise r test ./path/to/package`) over the full suite
- **Pre-commit**:
  - Before committing code:
    - Code must be formatted using `mise r fmt`
    - All tests must pass (any new or modified tests, plus `mise r test:sanity`)
    - Linting must not return any issues
- **Commit messages**: The repo follows [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/); prefixes like `feat: `, `fix: `, `chore: `, etc are required
- **Pull requests**: When creating pull requests, always include the following `gh pr new` flags:
  - `--draft`
  - `--assignee=@me`
  - `--base="$(git rev-parse --abbrev-ref "HEAD@{upstream}")"`

## Code Review
- **Review focus**: Code reviews should focus on (in order):
  - runtime safety
  - code correctness
  - language ecosystem idiomaticity
  - runtime performance and efficiency
  - code style
  - local codebase conventions
  - NOTE: This is not an exhaustive list; feel free to add additional foci as needed
- **Thoroughness**: ALWAYS be thorough, pedantic, and highly detailed
- **Interactivity**: ALWAYS perform reviews interactively, pausing after presenting every issue or piece of feedback and asking the user what they would like to do. Remind the user that you can make the changes for them if they'd like.
- **Ignored files**: Do not review any of the following:
  - .gitignore, or files matched by .gitignore
  - AGENTS.md (unless explicitly directed)
  - CLAUDE.md (unless explicitly directed)

## Code Style
- **General**: ALWAYS follow the Uber [Go Style Guide](https://raw.githubusercontent.com/uber-go/guide/refs/heads/master/style.md) whenever possible
- **Go fundamentals**: [Effective Go](https://go.dev/doc/effective_go), [Go Language Spec](https://tip.golang.org/ref/spec), [Go Memory Model](https://tip.golang.org/ref/mem)
- **Imports**: Standard library first, third-party (grouped), then local `github.com/nvidia-lpu/fake-lpu-operator/...`
  ```
  // Good
  import (
    "sync"
    "unsafe"

    "github.com/nvidia-lpu/minijinja"
    zlog "github.com/rs/zerolog/log"

    "github.com/nvidia-lpu/fake-lpu-operator/internal/must"
    "github.com/nvidia-lpu/fake-lpu-operator/internal/ptr"
    "github.com/nvidia-lpu/fake-lpu-operator/models"
  )

  // Bad
  import (
    "sync"
    "unsafe"

    "github.com/nvidia-lpu/minijinja"
    "github.com/nvidia-lpu/fake-lpu-operater/internal/must"
    "github.com/nvidia-lpu/fake-lpu-operater/internal/ptr"
    "github.com/nvidia-lpu/fake-lpu-operater/models"
    zlog "github.com/rs/zerolog/log"
  )
  ```
- **Import alias**: `zlog` for `github.com/rs/zerolog/log`
- **Line length**: 100 characters max (enforced by golines)
- **Naming**: CamelCase exports, _camelCase (prefixed by an underscore) unexported, descriptive names
- **Context naming**: Use consistent abbreviations for context variables:
  - `ctx` for `context.Context`
- **Error handling**: Early returns, `fmt.Errorf()` with `%w` verb
- **JSON tags**: `json:"field_name,omitempty"` (snake_case)
- **Struct tags**: snake_case for json/yaml/mapstructure
- **Constants**: TitleCase for exported constants
- **Variable declarations**: Group multiple consecutive variable declarations in `var` blocks, except when cuddling (e.g., `err := foo()` immediately followed by `if err != nil`)
  ```
  // Good
  var (
    foo = 123
    bar = true
  )

  things := make([]string, 0, foo)
  for range foo {
    things = append(things, ...)
  }

  // Bad
  foo := 123
  bar := true
  things := make([]string, 0, foo)

  for range foo {
    // ...
  }

  // Bad
  var (
    foo    = 123
    bar    = true
    things = make([]string, 0, foo)
  )

  for range foo {
    // ...
  }
  ```
- **Variable cuddling**: Outside of `var` blocks, cuddle (attach) variables to the blocks in which they're used
  ```
  // Good
  foo := make([]int, 0, 123)
  for range N {
    foo = append(foo, ...)
  }

  // Bad
  foo := make([]int, 0, 123)

  for range N {
    foo = append(foo, ...)
  }

  // Bad
  var (
    foo = make([]int, 0, 123)
    bar = true
  )

  for range N {
    foo = append(foo, ...)
  }
  ```
- **Conditionals**: Never use multiline conditionals; prefer cuddling variables with their usage
  ```
  // Good
  if err := doSomething(param1, param2, param3); err != nil {
    // ...
  }

  // Good
  err := doSomethingThatDoesntFitOnOneLine(
    param1,
    param2,
    param3,
    param4,
    param5,
  )
  if err != nil {
    // ...
  }

  // Bad
  if err := doSomethingThatDoesntFitOnOneLine(
    param1,
    param2,
    param3,
    param4,
    param5,
  ); err != nil {
    // ...
  }
  ```
- **Zero values**: Use `var` for zero value declarations (e.g., `var foo []T` not `foo := []T{}`), except maps which lack useful zero values
- **Line breaks**: When breaking into multiple lines, each item gets its own line, including closing parentheses/brackets
  ```
  // Good
  myMultiLineFunctionCall(
    foo,
    bar,
    baz,
  )

  // Bad
  myMultilineFunctionCall(foo,
    bar, baz)

  // Bad
  myMultilineFunctionCall(
    foo,
    bar,
    baz)

  // Bad
  myMultilineFunctionCall(foo, bar,
    baz)
  ```
- **Switch statements**: Use switch statements for multiple mutually-exclusive conditions, and always include a `default` case
- **Parameter and field types**: NEVER omit parameter or field types (sometimes referred to as "packing", e.g. `func foo(a, b, c string)`, `type foo { a, b, c string }`)
- **Branching and nesting**: Minimize branching/nesting, and prefer early returns when reasonable.

## Comments
- **Philosophy**: Code should be self-documenting through clear naming; minimize comments
- **When to comment**:
  - Exported symbols (required by language conventions)
  - Non-obvious business logic or design decisions
  - Surprising behavior or important constraints
- **When NOT to comment**:
  - Unexported functions with clear names
  - Implementation details obvious from the code
  - Restating what the function name already conveys
  - Standard patterns and obvious logic
- **Example**:
  ```
  // Good: Explains WHY we have this constraint
  // We retry up to 3 times because the upstream service has a known
  // issue where the first request occasionally fails with a 503.
  func retryUpstreamRequest(ctx context.Context) error { ... }

  // Good: Documents surprising behavior
  // Note: This modifies the input slice for performance reasons.
  func normalizeInPlace(data []float64) { ... }

  // Good: Godoc format - starts with symbol name, complete sentence
  // ProcessRequest handles incoming API requests and returns the response.
  func ProcessRequest(req *Request) (*Response, error) { ... }

  // Good: No comment needed - name and signature are clear
  func isValidEmail(email string) bool { ... }
  func calculateTotal(items []Item) float64 { ... }

  // Bad: Restates what the code obviously does
  // getUserByID gets a user by ID
  func getUserByID(id string) (*User, error) { ... }

  // Bad: Describes implementation details visible in the code
  // parseConfig reads the file, unmarshals JSON, and validates fields
  func parseConfig(path string) (*Config, error) { ... }
  ```

## Performance & Optimization
- **Priority**: Focus on reduce CPU time/cycles, memory allocations, and memory pressure
- **Efficiency**: Do not be unnecessarily wasteful with code - avoid allocations when practical
- **Memory**: Prefer object pooling, reuse buffers, avoid unnecessary allocations in hot paths
- **CPU**: Profile and optimize critical paths, avoid expensive operations in loops
- **Allocations**: Use `strings.Builder` over string concatenation, preallocate slices when size is known
- **Zero values**: Leverage useful zero values wherever possible - they're free and often exactly what you need

## Concurrency & Thread Safety
- **Mutex usage**: Always use `defer` to unlock mutexes unless absolutely necessary to manually unlock for performance reasons
- **Standard library helpers**: Prefer `slices` and `maps` package helpers where possible (e.g., `maps.Clone`, `slices.Clone`, `maps.All`)

## Code Organization
- **Logical blocks**: Organize code into logical blocks separated by blank lines:
  - Variable declarations and setup
  - Core logic/operations
  - Return statements
- **Consistent spacing**: Use blank lines to separate conceptually different operations within functions
- **Early returns**: Group error handling and early returns together, separated from main logic
- **Order of declarations**: For a given file, use the following order for declarations:
  - Package-level constants (grouped), with exported constants first and separated by a newline from unexported constants
  - Package-level variables (grouped), with exported variables first and separated by a newline from unexported variables
    - Errors should be separated (by a newline) from other vars
    - Interface assertions (e.g. `var _ Foo = (*bar)(nil)`) should be last, similarly separated by a newline
  - Exported types. For each exported type:
    - Type definition
    - Exported constructor(s)
    - Unexported constructor(s)
    - Exported receivers
    - Unexported receivers
  - Exported free functions
  - Unexported types. For each unexported type, follow the same pattern as exported types
  - Unexported free functions

## Testing
- **Unit tests**: Write proper, isolated unit tests for new code - no containers, network calls, or external dependencies
- **Coverage**: Aim for 100% unit test coverage, minimum 80% for new code
- **Testing libraries**: Use github.com/stretchr/testify for assertions and mocks; prefer the `require` subpackage over `assert`

## Filesystem modifications
- **Destructive operations**: NEVER perform destructive actions (removing files/directories, removing/changing existing lines within files) without asking for confirmation
- **Additive operations**: It is okay to perform creative actions (adding new lines to files, adding new files/directories) without prompting
- **Transparency**: ALWAYS display the EXACT command(s) that you will run to the user
