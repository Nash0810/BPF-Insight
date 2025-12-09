


# Contributing to BPF-Insight

This is an educational project built to production code standards. Contributions are welcome but should be approached with the understanding that this is a systems engineering learning exercise, not a commercial product.

## What This Project Is

- ✅ A deep technical case study in reverse-engineering the eBPF verifier
- ✅ Production-quality code for learning purposes
- ✅ A platform for exploring static analysis, abstract interpretation, and control flow analysis
- ❌ Not a commercial product or enterprise solution
- ❌ Not officially supported by kernel maintainers

## Getting Started

### Prerequisites

- Go 1.21+
- clang 14+ (for compiling test eBPF programs)
- Basic understanding of eBPF, the kernel verifier, and static analysis

### Setup

```bash
git clone https://github.com/Nash0810/BPF-Insight
cd BPF-Insight
make build
make test
```

## Before Contributing

**Familiarize yourself with:**
1. [ARCHITECTURE.md](./ARCHITECTURE.md) — Design decisions and data flow
2. [METHODOLOGY.md](./docs/METHODOLOGY.md) — How verifier constraints map to analysis
3. Existing code in `pkg/cfg/`, `pkg/verify/`, `pkg/parser/`

The project uses a clear pipeline:
```
ELF File → Parser → CFG Builder → Register Simulator → Analyzer → Report
```

## Code Quality Checklist

**Before submitting:**
```bash
make test       # All tests must pass
go fmt ./...    # Format code
go vet ./...    # Check for issues
```

**Code style:**
- Follow existing naming conventions
- Keep functions focused (single responsibility)
- Add comments for non-obvious logic
- Structure: types first, then helpers, then public methods

## Types of Contributions

### Bug Fixes
- Fix accuracy issues in existing rules
- Improve ELF parsing edge cases
- Correct CFG construction errors
- Better error messages

**Process:**
1. Describe the bug with test case
2. Fix the issue
3. Add regression test
4. Document the fix in PR

### New Verification Rules

**Important**: Understand the real verifier behavior first!

1. **Research:**
   - Read relevant kernel source (`kernel/bpf/verifier.c`)
   - Find test cases that trigger the constraint
   - Document why the pattern causes rejection

2. **Implement:**
   ```go
   Rules["my_rule"] = &Rule{
       Name:        "my_rule",
       Description: "What I'm checking",
       Enabled:     true,
       BlockCheckState: func(block *cfg.BasicBlock, ins parser.Instruction, state *RegState) []string {
           // Examine block, instruction, and register state
           if violatesConstraint {
               return []string{"Violation description"}
           }
           return []string{}
       },
   }
   ```

3. **Test:**
   - Create `.c` file in `test/validation/` that triggers the rule
   - Compile: `clang -O2 -target bpf -c test.c -o test.o`
   - Verify: `bpfva verify test.o --json`

4. **Document:**
   - Add section to [METHODOLOGY.md](./docs/METHODOLOGY.md)
   - Include example code (bad and good)
   - Explain the constraint

5. **Profile Assignment:**
   - Update `pkg/verify/profiles.go`
   - Assign to "strict"/"default"/"permissive"

### Performance Improvements

**Measurement required:**
```bash
# Baseline
/usr/bin/time -v bpfva batch test/compiled --recursive 2>&1 | grep elapsed

# After optimization
/usr/bin/time -v bpfva batch test/compiled --recursive 2>&1 | grep elapsed
```

**Document in PR:**
- Before/after times
- Test environment (CPU, RAM)
- Trade-offs (speed vs. accuracy)

### Documentation

- Update [EXAMPLES.md](./docs/EXAMPLES.md) for user-facing features
- Update [ARCHITECTURE.md](./ARCHITECTURE.md) for design changes
- Update [METHODOLOGY.md](./docs/METHODOLOGY.md) for analysis changes

### Tests

- Write table-driven tests for multiple scenarios
- Test both success and failure paths
- Aim for >80% coverage on new code

## Testing Guide

### Run Tests
```bash
go test ./...                      # All packages
go test ./pkg/cfg -v              # Verbose
go test ./pkg/verify -run TestXXX  # Specific test
```

### Integration Tests
```bash
make test       # Compile test programs and run analysis
sudo make validate  # Load with bpftool (requires root)
```

### Manual Testing
```bash
bpfva analyze test/compiled/simple.o
bpfva batch test/compiled --recursive
bpfva verify test/compiled/simple.o --json
```

## Commit Guidelines

Use clear, semantic commit messages:

```
[component] Brief description

Longer explanation of what changed and why.
Any relevant context or trade-offs.

Fixes #123
```

Examples:
```
[parser] Handle LD_IMM64 across section boundaries
[cfg] Improve loop detection with dominance analysis
[verify] Add map_no_null_check rule
[docs] Clarify register state propagation algorithm
```

## Pull Request Process

1. **Create feature branch:**
   ```bash
   git checkout -b feature/descriptive-name
   ```

2. **Make changes and test:**
   ```bash
   make test
   go test ./...
   ```

3. **Push and create PR:**
   - Reference related issues
   - Describe what changed and why
   - Note testing performed

4. **Address feedback** from review

5. **Merge** after approval

## Known Limitations & Opportunities

### Current Limitations
- Tested on Linux 5.15 only (kernel version-specific behavior not modeled)
- Register state tracking is conservative (may flag safe code as unsafe)
- No field offset tracking within structures
- Helper function side effects partially modeled

### Improvement Areas
- **Multi-kernel support**: Test against 5.10, 6.1, 6.6+
- **Field-sensitive analysis**: Track offsets within map values and stack
- **Improved state merging**: Better handling at loop headers
- **Performance profiling**: Find bottlenecks in large program analysis

## Questions?

- **Architecture questions**: See [ARCHITECTURE.md](./ARCHITECTURE.md)
- **Methodology questions**: See [METHODOLOGY.md](./docs/METHODOLOGY.md)
- **Usage questions**: See [EXAMPLES.md](./docs/EXAMPLES.md)
- **General questions**: Open a GitHub Discussion or Issue

---

**Thank you for contributing!** This project benefits from technical depth and practical feedback.
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
