# eBPF Verifier Complexity Analyzer

A static analysis tool that predicts eBPF program verifier rejection before kernel submission.

## Problem

The Linux kernel's eBPF verifier rejects programs that:
- Exceed complexity limits (1M instructions processed)
- Have unbounded loops or unclear bounds
- Perform unsafe memory operations
- Create excessive verification states

Developers waste hours debugging cryptic verifier errors with no guidance on where to fix issues.

## Solution

`bpfva` analyzes compiled eBPF bytecode and:
- Predicts rejection likelihood (0-100 score)
- Identifies complexity hotspots
- Provides actionable fix recommendations
- Visualizes control flow graphs

## Installation

### From Source
```bash
git clone https://github.com/Nash0810/BPF-Insight
cd BPF-Insight
make build
sudo cp bin/bpfva /usr/local/bin/
```

### Requirements
- Go 1.21+ (for building from source)
- clang 14+ (for compiling test programs)
- Graphviz (optional, for visualization rendering)

## Quick Start

### Analyze a Program
```bash
bpfva analyze my_program.o
```

### Generate Visualization
```bash
bpfva visualize my_program.o --render
```

### Compare Before/After Optimization
```bash
bpfva compare before.o after.o
```

### Batch Process Directory
```bash
bpfva batch ./ebpf_programs/ --recursive
```

## Usage

### analyze Command
```bash
bpfva analyze <file.o> [flags]

Flags:
  -j, --json              Output in JSON format
  -v, --verbose           Show detailed metrics
  --show-cfg              Generate CFG visualization
  -o, --output-dir string Directory for output files
  -n, --hotspots int      Number of hotspots to show (default 5)
  --no-viz                Skip visualization generation
```

### visualize Command
```bash
bpfva visualize <file.o> [flags]

Flags:
  -r, --render            Render PNG using Graphviz (dot)
```

### compare Command
```bash
bpfva compare <before.o> <after.o> [flags]

Flags:
  -f, --output-format string   Output format: text, json (default "text")
```

### batch Command
```bash
bpfva batch <directory> [flags]

Flags:
  -r, --recursive              Process subdirectories
  --pattern string            File glob pattern (default "*.o")
  -o, --output string         Output file for results
  --fail-threshold float      Exit with error if score > threshold
  -f, --output-format string  Output format: text, json (default "text")
```

## How It Works

### Analysis Process

1. **Bytecode Parsing**: Extracts eBPF instructions from ELF files
2. **CFG Construction**: Builds control flow graph from jump instructions
3. **Loop Detection**: Identifies back-edges via depth-first search
4. **Complexity Scoring**: Calculates score from:
   - Instruction count (40% weight)
   - Branch depth (25% weight)
   - Loop complexity (20% weight)
   - Branching factor (10% weight)
   - Helper calls (5% weight)
5. **Pattern Matching**: Detects known problematic constructs
6. **Recommendation Generation**: Provides actionable fixes

### Scoring Interpretation

- **0-40**: Low complexity (likely to pass verifier)
- **41-70**: Medium complexity (may require optimization)
- **71-90**: High complexity (likely to fail)
- **91-100**: Very high complexity (will almost certainly fail)

### Detected Patterns

1. **Unbounded Loops**: Loops without explicit bounds
2. **Complex Pointer Arithmetic**: Excessive pointer calculations
3. **Excessive Branching**: Blocks with >3 successors
4. **Helper Calls in Loops**: Inefficient helper placement
5. **Missing Bounds Checks**: Packet access without validation

## Examples

### Example 1: Simple Analysis
```bash
$ bpfva analyze examples/xdp_drop.o

eBPF Verifier Complexity Analysis
==================================
File: examples/xdp_drop.o
Section: xdp

Metrics:
  Instructions:  2
  Jumps:         0
  Helper Calls:  0

Complexity Score: 0.1 / 100
Prediction: LIKELY PASS

Analysis: This program has minimal complexity.
```

### Example 2: With Recommendations
```bash
$ bpfva analyze examples/complex_filter.o --verbose

[... metrics output ...]

Complexity Score: 82.5 / 100
Prediction: LIKELY FAIL

Recommendations:
  🔴 HIGH: Unbounded loop detected at Block 8 (insn 156)
     └─ Add explicit bound check. Use #pragma unroll with fixed count.
  
  🟠 MEDIUM: Excessive branching in Block 15 (4 successors)
     └─ Consider using BPF maps for lookup tables.
  
  🟡 LOW: Helper call inside loop at insn 234
     └─ Move helper call outside loop if possible.
```

### Example 3: Visualization
```bash
$ bpfva visualize examples/nested_loops.o --render

Generating CFG visualization...
  └─ Parsed 1,024 instructions
  └─ Built CFG with 45 basic blocks
  └─ Detected 2 loops
  └─ Generated DOT file: nested_loops_cfg.dot
  └─ Rendering to PNG...
  └─ Saved: nested_loops_cfg.png
```

## Limitations

### What This Tool Does NOT Do

1. **Does not replace the kernel verifier** - This tool provides predictions, not guarantees
2. **Does not modify programs** - It only analyzes and recommends
3. **Does not verify memory safety** - It focuses on complexity, not correctness
4. **Does not require kernel access** - All analysis is static

### Known Issues

- **False Positives**: Some programs may score high but still pass
- **False Negatives**: Complex verifier logic may reject low-scoring programs
- **Kernel Version Differences**: Verifier limits vary across kernel versions
- **Heuristic-Based**: Scoring uses approximations, not exact verifier logic

### Accuracy

Based on validation against real programs:
- **Correct predictions**: ~80%
- **Uncertain (MAY_PASS)**: ~15%
- **Incorrect predictions**: ~5%

## Architecture

```
┌─────────────────────────────────────────────────┐
│                 CLI Interface                    │
│              (cobra commands)                    │
└──────────────┬──────────────────────────────────┘
               │
       ┌───────┴────────┐
       │                │
┌──────▼──────┐  ┌──────▼──────┐
│   Parser    │  │  Analyzer   │
│  (ELF/BPF)  │  │   (CFG)     │
└──────┬──────┘  └──────┬──────┘
       │                │
       └────────┬───────┘
                │
    ┌───────────▼────────────┐
    │   Recommendation       │
    │      Engine            │
    └───────────┬────────────┘
                │
    ┌───────────▼────────────┐
    │    Output Formatter     │
    │   (Text/JSON/DOT)      │
    └────────────────────────┘
```

## Documentation

- **[README.md](README.md)** - Main documentation (this file)
- **[docs/METHODOLOGY.md](docs/METHODOLOGY.md)** - Technical deep dive on analysis methodology
- **[docs/EXAMPLES.md](docs/EXAMPLES.md)** - Detailed usage examples and use cases

## Development

### Project Structure
```
BPF-Insight/
├── cmd/              # CLI commands
├── pkg/
│   ├── parser/       # ELF and instruction parsing
│   ├── analyzer/     # CFG and complexity analysis
│   ├── cfg/          # Control flow graph construction
│   └── verify/       # Pattern detection and rules
├── test/
│   ├── programs/     # Test C files
│   └── compiled/     # Compiled test programs
├── scripts/          # Validation and build scripts
└── docs/             # Additional documentation
```

### Running Tests
```bash
make test
```

### Building from Source
```bash
make build
```

### Validating Predictions
```bash
# Requires root for bpftool
sudo make validate
```

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure `make test` passes
5. Submit a pull request

## License

MIT License - see LICENSE file

## Acknowledgments

- Linux kernel eBPF verifier team
- Cilium eBPF library maintainers
- CNCF eBPF projects (Falco, Tetragon)

