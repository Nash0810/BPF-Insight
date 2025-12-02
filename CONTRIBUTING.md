
# Contributing to BPF-Insight

Thank you for your interest in contributing to BPF-Insight. This document provides guidelines and procedures for contributing to the project.

## Getting Started

### Prerequisites

- Go 1.21 or later
- clang 14 or later
- libbpf-dev
- Graphviz (optional, for visualization)
- git

### Build from Source

```bash
git clone https://github.com/Nash0810/BPF-Insight
cd BPF-Insight
make build
```

### Run Tests

```bash
make test
```

## Development Workflow

### 1. Fork and Clone

```bash
git clone https://github.com/YOUR-USERNAME/BPF-Insight
cd BPF-Insight
git remote add upstream https://github.com/Nash0810/BPF-Insight
```

### 2. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

Use descriptive branch names:
- `feature/add-new-rule` for new features
- `fix/null-pointer-check` for bug fixes
- `docs/update-readme` for documentation
- `test/add-coverage` for tests

### 3. Make Changes

- Write clear, idiomatic Go code
- Follow existing code style and organization
- Add tests for new functionality
- Update documentation as needed

### 4. Commit Changes

Use semantic commit messages:

```
feat: add shift amount validation rule

- Detect shifts >= 32 bits on scalars
- Classify as MEDIUM severity
- Add test coverage for edge cases
```

Format: `<type>: <subject>`

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `test`: Test additions/changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build/tooling changes

### 5. Test Your Changes

```bash
make build      # Compile
make test       # Run tests
make clean      # Clean artifacts
```

Ensure all tests pass before submitting.

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub with:
- Clear description of changes
- Reference to related issues
- Test coverage information
- Performance impact (if applicable)

## Code Style

### Go Code Guidelines

- Format code with `gofmt`
- Use meaningful variable names
- Keep functions focused and small
- Add comments for exported functions
- Document complex logic inline

### File Organization

- Place related functionality in the same package
- Keep `pkg/` modular and single-responsibility
- Use clear package names (analyzer, parser, cfg, verify)

### Comments

Comment exported functions and types:

```go
// Analyze performs static complexity analysis on eBPF bytecode.
// It takes a file path and returns a ComplexityReport or error.
func Analyze(filePath string) (*ComplexityReport, error) {
```

## Testing Requirements

### Adding Tests

- Add test files alongside implementation (e.g., `analyzer_test.go`)
- Use table-driven tests for multiple scenarios
- Test both success and failure cases
- Aim for >80% code coverage

### Test File Organization

```go
package analyzer

import "testing"

func TestAnalyze(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    int
        wantErr bool
    }{
        {"valid program", "simple.o", 10, false},
        {"invalid file", "missing.o", 0, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Documentation

### README Updates

When adding features, update README.md:
- Add command examples to "Command Reference"
- Document new flags/options
- Update architecture if needed

### Inline Documentation

- Use clear, technical language
- Avoid jargon without explanation
- Document assumptions and limitations

### Release Notes

For user-facing changes, document in appropriate changelog.

## Pull Request Process

1. **Ensure tests pass**: `make test`
2. **Build successfully**: `make build`
3. **Code review**: Address feedback promptly
4. **Maintainer approval**: Wait for maintainer sign-off
5. **Merge**: PR will be merged to main branch

## Areas for Contribution

### High Priority

- Additional eBPF program types (TC, kretprobe)
- Enhanced helper profiles for different kernel versions
- Improved register-state analysis

### Medium Priority

- Architecture support (ARM64, RISC-V)
- Performance optimizations
- Documentation improvements

### Low Priority

- UI/UX enhancements
- Example programs
- Visualization improvements

## Reporting Issues

When reporting bugs, include:

- **Title**: Clear, descriptive
- **Description**: What happened and what was expected
- **Reproduction steps**: How to reproduce the issue
- **Environment**: Go version, OS, kernel version
- **Attachments**: Test program (if applicable)

Example:

```
Title: Analyzer crashes on large programs

Description:
When analyzing programs with >1000 instructions, the analyzer crashes.

Steps to reproduce:
1. Compile a large test program
2. Run: bpfva analyze large_program.o
3. Observe crash

Environment:
- Go 1.21
- Linux 5.15
- Kernel 5.15.0-56-generic
```

## Licensing

By contributing, you agree that your contributions will be licensed under the Apache License 2.0. Ensure any new files include the license header if required.

## Questions or Discussions

- **General questions**: Use GitHub Discussions
- **Bug reports**: Use GitHub Issues
- **Feature requests**: Use GitHub Issues with `[FEATURE]` tag

## Code of Conduct

Contributors are expected to:
- Be respectful and inclusive
- Provide constructive feedback
- Follow project conventions
- Report violations to maintainers

## Recognition

Contributors will be recognized in:
- Project README
- Release notes for significant contributions
- GitHub contributors page

Thank you for contributing to BPF-Insight!

---

For questions, please open an issue or start a discussion on GitHub.
