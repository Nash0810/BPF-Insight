# BPF-Insight: eBPF Verifier Complexity Analysis

A static analysis tool for predicting eBPF program verifier acceptance and rejection prior to kernel submission.

**v1.0.0 Release - Validation Accuracy: 100% (15/15 test cases)**

## Overview

The Linux kernel eBPF verifier enforces constraints on program structure and resource consumption including instruction processing limits (1 million per program), loop boundedness requirements, memory access validation, and state space explosion prevention. Determining verifier acceptance of programs with complex control flow patterns presents significant analysis challenges. Manual debugging of verifier rejection errors often provides limited guidance on root causes.

BPF-Insight provides:
- Complexity score predictions (0-100 scale)
- Pattern-based root cause identification
- Control flow graph analysis and visualization
- Batch processing capabilities for program suites
- Programmatic output formats (text, JSON, DOT)

## Building and Installation

### From Source
```bash
git clone https://github.com/Nash0810/BPF-Insight
cd BPF-Insight
make build
sudo make install
```

### Requirements
- Go 1.21 or later
- clang 14 or later (for compiling eBPF test programs)
- libbpf-dev (for linking)
- Graphviz (optional, for visualization rendering)

## Quick Start

### Basic Analysis
```bash
bpfva analyze my_program.o
```

### Control Flow Graph Visualization
```bash
bpfva visualize my_program.o --render
```

### Comparative Analysis
```bash
bpfva compare before.o after.o
```

### Batch Analysis
```bash
bpfva batch ./programs --recursive
```

## Release Notes - v1.0.0

**Status**: Production Release  
**Validation**: 100% accuracy on 15 confident test cases

### Features

- **Custom eBPF Instruction Decoder**: Complete support for LD_IMM64 (64-bit immediate instructions)
- **Register-State Tracking**: Conservative taint propagation for pointer detection
- **Verifier Rule Engine**: 11+ pattern detections including pointer arithmetic, R10 writes, null check validation
- **Severity-Based Scoring**: CRITICAL, HIGH, MEDIUM, LOW classifications with weighted impact
- **Control Flow Graph Visualization**: CFG generation and rendering support
- **Batch Processing**: Multi-program analysis with aggregate reporting
- **Programmatic Output**: JSON format for integration with external tools

### Improvements in v1.0.0

- Refined penalty scoring methodology (15/10/5/2 point scale)
- Enhanced BTF detection logic
- Shift amount validation for boundary detection
- Conservative uncertainty categorization
- Comprehensive ELF section handling

### Installation Options

**Pre-built Binary**
```bash
wget https://github.com/Nash0810/BPF-Insight/releases/download/v1.0.0/bpfva-linux-amd64
chmod +x bpfva-linux-amd64
sudo mv bpfva-linux-amd64 /usr/local/bin/bpfva
```

**Build from Source**
```bash
git clone https://github.com/Nash0810/BPF-Insight
cd BPF-Insight
make build
sudo make install
```

## Command Reference

### analyze - Program Analysis
```bash
bpfva analyze <file.o> [flags]

Flags:
  -j, --json              JSON output format
  -v, --verbose           Detailed metrics
  --show-cfg              Generate CFG visualization
  -o, --output-dir        Output directory for generated files
  -n, --hotspots          Number of complexity hotspots (default: 5)
  --no-viz                Disable visualization generation
```

### visualize - Control Flow Graph Rendering
```bash
bpfva visualize <file.o> [flags]

Flags:
  -r, --render            Render to PNG using Graphviz
```

### compare - Differential Analysis
```bash
bpfva compare <before.o> <after.o> [flags]

Flags:
  -f, --output-format     Output format: text, json (default: text)
```

### batch - Bulk Program Analysis
```bash
bpfva batch <directory> [flags]

Flags:
  -r, --recursive         Process subdirectories
  --pattern               File glob pattern (default: *.o)
  -o, --output            Output file for results
  --fail-threshold        Error threshold for exit code
  -f, --output-format     Output format: text, json (default: text)
```

## Analysis Methodology

### Process Overview

1. **ELF Parsing**: Instruction extraction from compiled object files
2. **Control Flow Graph Construction**: Block identification and edge analysis from jump instructions
3. **Loop Analysis**: Back-edge detection and structural classification
4. **Complexity Scoring**: Multi-factor calculation including instruction count, branch depth, loop nesting, branching factor, and helper invocation count
5. **Pattern Matching**: Detection of known problematic constructs
6. **Result Generation**: Score normalization and recommendation synthesis

### Scoring Model

Complexity scores range from 0-100 and are calculated as:
- **0-24**: Low complexity, likely verifier acceptance
- **25-49**: Moderate complexity, uncertain classification
- **50-74**: High complexity, likely verifier rejection
- **75-100**: Very high complexity, probable rejection

### Detection Patterns

The analysis engine detects the following patterns:

1. Unbounded Loops - Loops without explicit iteration bounds
2. Complex Pointer Arithmetic - Multiple consecutive pointer operations
3. High Branch Factor - Basic blocks with excessive successors
4. Helper Call Inefficiency - Helper invocations within loop constructs
5. Bounds Check Omission - Memory access without prior validation

## Example Output

### Basic Analysis
```
$ bpfva analyze examples/simple.o

eBPF Program Analysis Report
===========================
File: examples/simple.o
Section: xdp

Metrics:
  Instructions:     12
  Basic Blocks:     2
  Helper Calls:     0
  Loops Detected:   0

Complexity Score: 5.2 / 100
Prediction: LIKELY_PASS

Analysis: Program structure indicates high probability of verifier acceptance.
```

### Complex Program Analysis
```
$ bpfva analyze examples/complex.o --verbose

[... metrics output ...]

Complexity Score: 68.4 / 100
Prediction: LIKELY_FAIL

Detected Issues:
  CRITICAL: Unbounded loop at block 3
    Remediation: Add explicit iteration limit using pragma unroll
  HIGH: Pointer arithmetic on inferred register
    Remediation: Validate pointer bounds prior to arithmetic operations
```

### Batch Processing
```
$ bpfva batch ./programs --recursive --output results.json

Processing 24 programs...
  Completed: 24
  Failed: 0
  
High-Risk Programs: 3
  - complex_filter.o (score: 78.5)
  - data_processing.o (score: 72.1)
  - packet_handler.o (score: 70.3)

Average Complexity: 35.2
Median Complexity: 28.1
```

## Limitations and Scope

### Tool Boundaries

- **Prediction Basis**: Results are derived from static analysis heuristics and may not align with kernel verifier behavior in all cases
- **No Program Modification**: Tool provides analysis only; remediation is user-directed
- **Static Analysis**: Does not verify memory safety or semantic correctness
- **Kernel Independence**: Operates without kernel or verifier access

### Known Constraints

- **False Positive Potential**: Programs scoring high may still achieve verifier acceptance in specific kernel versions
- **False Negative Potential**: Complex verifier state interactions may reject programs scoring lower than expected
- **Version Sensitivity**: Verifier behavior and limits vary across kernel versions
- **Heuristic-Based**: Scoring employs approximations rather than exact verifier simulation

### Validation Statistics

Based on comprehensive testing against actual programs:
- Confident predictions: ~80% accuracy
- Uncertain classifications (MAY_PASS): ~15%
- Prediction discrepancies: ~5%

## Architecture

```
┌────────────────────────────────────┐
│       Command Line Interface       │
│         (Cobra Framework)          │
└──────────────┬─────────────────────┘
               │
       ┌───────┴───────┐
       │               │
   ┌───▼────┐    ┌────▼───┐
   │ Parser │    │Analyzer │
   │(ELF)   │    │ (CFG)   │
   └───┬────┘    └────┬───┘
       │              │
       └────┬─────────┘
            │
    ┌───────▼──────────┐
    │ Rule Engine      │
    │ (Pattern Match)  │
    └───────┬──────────┘
            │
    ┌───────▼──────────┐
    │ Output Formatter │
    │(Text/JSON/DOT)   │
    └──────────────────┘
```

## Project Organization

```
BPF-Insight/
├── cmd/                  # CLI implementation
│   ├── main.go
│   ├── analyze.go
│   ├── verify.go
│   ├── visualize.go
│   ├── compare.go
│   └── batch.go
├── pkg/
│   ├── analyzer/         # Complexity scoring
│   │   ├── complexity.go
│   │   └── comparator.go
│   ├── cfg/              # Control flow graph
│   │   ├── cfg.go
│   │   ├── dot.go
│   │   └── score.go
│   ├── parser/           # ELF and instruction parsing
│   │   ├── decode.go
│   │   ├── elf.go
│   │   └── instruction.go
│   ├── utils/            # Utility functions
│   │   └── json.go
│   └── verify/           # Pattern detection
│       ├── rules.go
│       ├── verify.go
│       ├── regstate.go
│       ├── ruleregistry.go
│       ├── aliases.go
│       └── profiles.go
├── test/
│   ├── programs/         # eBPF source files
│   ├── compiled/         # Compiled test objects
│   └── validation/       # Validation test programs
├── scripts/
│   ├── validate.sh       # Test harness
│   └── report_warnings.sh
├── Makefile              # Build automation
└── go.mod / go.sum       # Dependency management
```

## Development

### Building
```bash
make build
```

### Testing
```bash
make test
```

### Validation Against Kernel Verifier
```bash
# Requires elevated privileges for bpftool access
sudo make validate
```

### Installation
```bash
make install
```

## Contributing

Contributions are welcome. Please ensure:
- All tests pass (`make test`)
- Code follows project conventions
- New functionality includes test coverage
- Pull requests reference related issues

### Future Enhancements

- Extended register-state analysis with field offset tracking
- Kernel version-specific helper profiles
- Architecture support expansion (ARM64, RISC-V)
- Additional program type support (TC, kretprobe)
- Improved pattern detection through machine learning

