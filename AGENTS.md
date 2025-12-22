# Repository Guidelines

This document provides comprehensive guidelines for contributing to the `go.lsp.dev/protocol` repository, which implements the Language Server Protocol (LSP) specification in Go.

## Project Overview

This package provides a Go implementation of the Language Server Protocol (LSP) specification. It contains structs that map directly to the wire format of LSP, serving as a literal transcription with modifications necessary for Go compatibility.

**Key characteristics:**
- Literal transcription of LSP specification with unmodified comments
- Names uppercased for Go export requirements
- JSON tags for correct field name mapping
- Optional fields marked with `omitempty`
- Nullable fields implemented as pointers

## Project Structure

```
.
├── *.go                    # Core LSP protocol implementations (root package)
├── *_test.go              # Unit tests for protocol implementations
├── internal/              # Internal packages (code generation tools)
│   └── lspgen/           # LSP code generator
├── tools/                 # Build and development tools
├── docs/                  # Documentation and references
├── hack/                  # Build scripts and utilities
│   ├── boilerplate/      # License headers and templates
│   └── make/             # Make helpers
├── .circleci/            # CircleCI configuration
├── .github/              # GitHub workflows (CodeQL security scanning)
└── vendor/               # Vendored dependencies
```

**File organization by LSP domain:**
- `base.go`, `basic.go` - Fundamental types and structures
- `general.go` - General protocol methods
- `language.go` - Language feature implementations
- `text.go` - Text document synchronization
- `window.go` - Window-related operations
- `workspace.go` - Workspace management
- `capabilities_*.go` - Client and server capabilities
- `diagnostics.go` - Diagnostic reporting
- `progress.go` - Progress reporting
- `callhierarchy.go` - Call hierarchy support
- `semantic_token.go` - Semantic token support

## Build and Development Commands

### Essential Commands

```bash
# Run tests with race detection
make test

# Run tests with coverage
make coverage

# Format code (goimportz + gofumpt)
make fmt

# Run linters (golangci-lint)
make lint

# Clean build artifacts
make clean

# Install development tools
make tools

# Install specific tool
make tools/gotestsum
```

### Testing

```bash
# Run all tests with race detection
CGO_ENABLED=1 make test

# Run specific test function
GO_TEST_FUNC=TestCancelParams make test

# Generate coverage report
make coverage

# View coverage in browser
go tool cover -html=coverage.out
```

### Formatting and Linting

```bash
# Format all Go files (runs goimportz then gofumpt)
make fmt

# Run all linters
make lint

# Run specific linter check
make lint/golangci-lint
```

### Utilities

```bash
# Find TODO/BUG/XXX/FIXME comments
make todo

# Find nolint pragmas
make nolint

# Print Makefile variable value
make env/GO_FLAGS
```

## Coding Style and Conventions

### Go Version

- **Minimum required:** Go 1.24.0
- Follow Go 1.18+ best practices (generics, any, etc.)

### Code Style

1. **Follow Google Go Style Guide:**
   - [Go Style Guide](https://google.github.io/styleguide/go/guide)
   - [Go Style Decisions](https://google.github.io/styleguide/go/decisions)
   - [Go Style Best Practices](https://google.github.io/styleguide/go/best-practices)

2. **Formatting:**
   - Use `gofmt -s` for simplification
   - Use `gofumpt -extra` for stricter formatting
   - Use `goimportz` for import organization with local prefix `go.lsp.dev/protocol`

3. **Import organization:**
   ```go
   import (
       // Standard library
       "context"
       "fmt"

       // External dependencies
       "github.com/segmentio/encoding/json"
       "go.lsp.dev/jsonrpc2"

       // Local packages
       "go.lsp.dev/protocol/internal/..."
   )
   ```

4. **Documentation:**
   - All exported symbols must have godoc comments
   - **Comments must end with a period**
   - Comments should explain the "why," not just the "what"

5. **JSON marshaling:**
   - Use `github.com/segmentio/encoding/json` (not standard `encoding/json`)
   - Use `omitzero` tag for optional fields (spec convention)

### Naming Conventions

- **Types:** PascalCase (e.g., `CancelParams`, `ProgressToken`)
- **Functions/Methods:** camelCase for private, PascalCase for exported
- **Constants:** PascalCase for exported, camelCase for private
- **Variables:** camelCase (short names for limited scope, descriptive for wider scope)
- **Test functions:** `TestTypeName` or `TestFunctionName`
- **Error types:** Suffix with `Error` (e.g., `ResponseError`)
- **Error sentinel values:** Prefix with `Err` (e.g., `ErrInvalidRequest`)

### Code Organization

- Keep related functionality together by LSP domain
- One type and its methods per logical section
- Tests in separate `*_test.go` files matching the source file name
- Use file-level comments to describe the LSP section being implemented

## Testing Guidelines

### Test Framework

- **Framework:** Standard `testing` package with `github.com/google/go-cmp/cmp`
- **Runner:** `gotestsum` for better output formatting
- **Coverage target:** Aim for >75% (commendable), >90% (exemplary)

### Test Structure

All tests must follow this structure:

```go
func TestTypeName(t *testing.T) {
    t.Parallel()

    // Define test data
    const want = `{"field":"value"}`
    wantType := TypeName{
        Field: "value",
    }

    t.Run("Marshal", func(t *testing.T) {
        t.Parallel()

        tests := map[string]struct {
            name           string
            field          TypeName
            want           string
            wantMarshalErr bool
            wantErr        bool
        }{
            "success: basic case": {
                field:          wantType,
                want:           want,
                wantMarshalErr: false,
                wantErr:        false,
            },
            "error: invalid input": {
                field:          TypeName{},
                want:           "",
                wantMarshalErr: true,
                wantErr:        true,
            },
        }

        for testName, tt := range tests {
            tt := tt
            t.Run(testName, func(t *testing.T) {
                t.Parallel()

                got, err := json.Marshal(&tt.field)
                if (err != nil) != tt.wantMarshalErr {
                    t.Fatal(err)
                }

                if diff := cmp.Diff(tt.want, string(got)); (diff != "") != tt.wantErr {
                    t.Errorf("%s: wantErr: %t\n(-want +got)\n%s", testName, tt.wantErr, diff)
                }
            })
        }
    })
}
```

### Test Naming Conventions

- **Test cases:** Use `map[string]struct{}` with descriptive keys
- **Case naming:** Format as `"status: description"`
  - `"success: basic case"`
  - `"success: with nil value"`
  - `"error: empty input"`
  - `"error: invalid type"`

### Test Requirements

1. **Parallel execution:** Use `t.Parallel()` at test and subtest level
2. **Table-driven tests:** Use map-based test tables for all tests
3. **Assertions:** Use `cmp.Diff()` for comparisons, not `reflect.DeepEqual()`
4. **Error handling:** Check both error occurrence and expected state
5. **Coverage:** Test both Marshal and Unmarshal for JSON types
6. **Context usage:** Use `t.Context()` when context is needed (not `context.Background()`)

### Test Exclusions

The following checks are disabled for test files (`.golangci.yml`):
- `errcheck` - Error checking not always necessary in tests
- `funlen` - Test functions can be longer
- `gocognit`, `gocyclo` - Complexity rules relaxed
- `gosec` - Security checks less critical in tests
- `lll` - Line length limits relaxed

## Linting Configuration

The repository uses `golangci-lint` with an extensive set of enabled linters. Key configurations:

### Complexity Limits

- **Cyclomatic complexity:** 30 (gocyclo, cyclop)
- **Cognitive complexity:** 30 (gocognit)
- **Function length:** 120 lines / 60 statements
- **Maintainability index:** 15
- **Naked return limit:** 30 lines

### Specific Linter Settings

- **gofumpt:** Extra rules enabled
- **goimports:** Local prefix `go.lsp.dev/protocol`
- **govet:** All checks enabled except `fieldalignment`
- **varnamelen:** Minimum 1 character, max distance 5
- **godot:** Comments must end with period

### Excluded Warnings

- Shadow warnings for `err`, `ok`, `ctx` variables
- `continue` with no blank line before

## Commit and Pull Request Guidelines

### Commit Message Format

Based on Git history analysis, follow these conventions:

```
<scope>: <imperative description>

<optional body>

<optional footer>
```

**Scope examples:**
- `all` - Changes across multiple files
- `go.mod`, `Makefile`, `.circleci` - Configuration changes
- `golangci-lint` - Linter configuration
- Specific package names - Package-specific changes

**Common patterns:**
```
all: fix lint issues
all: run gofumpt
Makefile: remove fmt target deps on lint
golangci-lint: update lint config
go.mod: update dependency packages to latest
tools: update tools to latest
context: fix context key (#46)
```

### Commit Requirements

1. **Message format:**
   - Start with scope followed by colon
   - Use imperative mood ("fix" not "fixed", "add" not "added")
   - Keep first line under 72 characters
   - Reference issue numbers with `(#123)` suffix

2. **GPG signing:**
   ```bash
   git commit --gpg-sign --signoff -m "scope: description"
   ```

3. **Code quality:**
   - All tests must pass
   - No linting errors
   - Run `make fmt` before committing
   - Update tests for behavior changes

### Pull Request Guidelines

1. **Before submitting:**
   - Run full test suite: `make test`
   - Check coverage: `make coverage`
   - Verify formatting: `make fmt && git diff --exit-code`
   - Run linters: `make lint`

2. **PR description should include:**
   - Summary of changes
   - Related issue numbers
   - Breaking changes (if any)
   - Testing performed

3. **Review process:**
   - Ensure CI passes (CircleCI, CodeQL)
   - Address review comments
   - Keep commits focused and logical

## License and Copyright

- **License:** BSD-3-Clause
- **Copyright holder:** The Go Language Server Authors
- **Header format:**
  ```go
  // SPDX-FileCopyrightText: 2021 The Go Language Server Authors
  // SPDX-License-Identifier: BSD-3-Clause
  ```

All new files must include this header. Use `hack/boilerplate/` templates for consistency.

## Continuous Integration

### CircleCI

The project uses CircleCI for:
- Running tests with race detection
- Code coverage collection and upload to Codecov
- Linting verification
- Multi-version Go testing

**Configuration:** `.circleci/config.yml`

### GitHub Actions

- **CodeQL:** Security vulnerability scanning
- **Configuration:** `.github/workflows/codeql.yml`

## Dependencies

### Core Dependencies

- `github.com/google/go-cmp` - Test assertions
- `github.com/segmentio/encoding` - Fast JSON encoding
- `go.lsp.dev/jsonrpc2` - JSON-RPC 2.0 implementation
- `go.lsp.dev/uri` - URI handling
- `go.uber.org/zap` - Logging
- `mvdan.cc/gofumpt` - Code formatting

### Development Tools

Managed in `tools/` directory:
- `github.com/golangci/golangci-lint` - Linting
- `mvdan.cc/goimportz` - Import formatting
- `mvdan.cc/gofumpt` - Code formatting
- `gotest.tools/gotestsum` - Test runner

## LSP-Specific Considerations

### Protocol Fidelity

- Maintain exact correspondence with LSP specification
- Preserve original specification comments
- Document any deviations or Go-specific adaptations

### JSON Handling

- All protocol types must support JSON marshaling/unmarshaling
- Test both directions for every type
- Handle optional fields correctly (pointers + omitzero)
- Support `string | number` union types appropriately

### Versioning

This repository tracks LSP specification versions via Git worktrees. The current directory represents LSP version 3.17.0.

### Related Resources

- [LSP Specification](https://microsoft.github.io/language-server-protocol/)
- [Implementation References](docs/implementation-references.md)

## Getting Help

- **Issues:** File bug reports and feature requests on GitHub
- **Discussions:** Use GitHub discussions for questions
- **Documentation:** See `docs/` directory for additional resources

## Quick Start Checklist

For new contributors:

- [ ] Clone repository
- [ ] Install Go 1.24.0+
- [ ] Run `make tools` to install development tools
- [ ] Run `make test` to verify setup
- [ ] Run `make fmt` before committing
- [ ] Run `make lint` to check code quality
- [ ] Add tests for all new code
- [ ] Ensure coverage doesn't decrease
- [ ] Follow commit message conventions
- [ ] Sign commits with GPG

---

*This guide is maintained by the Go Language Server Authors. Last updated: 2025-12-22*
