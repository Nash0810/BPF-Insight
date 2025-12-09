# BPF-Insight: eBPF Verifier Complexity Analysis

A reverse-engineering project that models the Linux kernel eBPF verifier's acceptance/rejection logic through static analysis, control flow graph construction, and abstract interpretation of register states.

**Status**: Educational project built to production code standards  
**Validation**: 80%+ accuracy on binary predictions (PASS/FAIL); tested against kernel verifier on 15 representative programs

## What This Is

BPF-Insight reconstructs verifier behavior by analyzing:
- **Control flow structure** via CFG construction (block leaders, back-edge detection)
- **Register state evolution** using abstract interpretation (`R0`-`R10` tracking)
- **Known rejection patterns** through a modular rule engine (11 pattern detections)

The goal is to give developers *before-submission feedback* on whether their eBPF programs will likely pass the kernel verifier, without needing kernel access or trial-and-error compilation.

## Key Design Decisions

**Zero-Dependency ELF Parser**: Rather than invoke `llvm-objdump` or parse external tools, BPF-Insight includes a custom ELF parser written from scratch in Go (`pkg/parser/elf.go`). This lets the tool work in isolated environments and avoids shell execution overhead.

**Abstract Interpretation for Register Tracking**: Instead of concrete execution simulation (which would require kernel context), the verifier simulator uses conservative taint propagation. Registers holding pointers are tracked as "pointer-type" with bounded arithmetic operations flagged as errors. This is sufficient to catch ~80% of real verifier rejections without simulating actual execution.

**CFG-First, Not AST-First**: The analysis builds a control flow graph directly from instructions rather than constructing an intermediate representation. This reduces memory overhead and keeps analysis parallelizable if needed later.

**Modular Rule Engine**: Each verifier constraint is implemented as a separate rule (e.g., "no writes to R10 frame pointer", "map lookups must be null-checked"). Rules can be enabled/disabled via profiles ("strict", "default", "permissive") for different use cases.

## Building and Installation

### From Source
```bash
git clone https://github.com/Nash0810/BPF-Insight
cd BPF-Insight
make build
sudo make install
```

### Requirements
- Go 1.21+
- clang 14+ (for compiling eBPF test programs)
- Optional: Graphviz `dot` command (for CFG rendering)

## Technical Highlights

### 1. Control Flow Graph Construction with Loop Detection

The CFG builder (`pkg/cfg/cfg.go`) uses the classic leader-based block identification algorithm:

```
Input: Sequence of instructions
Step 1: Identify block leaders (entry point, jump targets, post-jump fallthrough)
Step 2: Partition instructions into basic blocks
Step 3: Connect blocks with edges (jump, fallthrough, back-edge)
Step 4: Run DFS to detect back-edges (indicating loops)
```

This enables detection of unbounded loops—a primary verifier rejection cause. The DFS-based loop detection runs in O(V+E) time where V=blocks, E=edges.

### 2. Abstract Interpretation of Register States

The register state tracker (`pkg/verify/regstate.go`) maintains:
- Register type (UNKNOWN, CONST_VAL, POINTER, STACK_PTR, MAP_PTR)
- Value bounds (min/max for constants)
- Offset tracking (for pointer arithmetic)

At each instruction, state is updated conservatively:
- Pointer + Arithmetic → Invalidates pointer bounds → Flagged as error
- ALU on unknown value → State marked UNKNOWN
- Merge of states at block entry → Union of possibilities

This allows the verifier simulator to reject ~80% of programs that would actually fail the kernel verifier, without concrete execution.

### 3. Rule Engine with Profile System

The verification engine (`pkg/verify/rules.go`) registers 11+ rules:
- pointer_arithmetic, helper_chain, stack_var_offset, write_r10
- map_no_null_check, map_update_no_key_check, unknown_helper
- pointer_bitwise_shift, missing_btf, suspicious_shifts, high_complexity

Rules can be enabled/disabled via profiles:
```bash
bpfva verify program.o --profile strict    # All rules enabled
bpfva verify program.o --profile permissive # Loose checking
```

### 4. Custom ELF Parser (Zero External Dependencies)

Rather than shell out to `objdump` or `llvm-objdump`, the ELF parser directly reads section headers and extracts `.text` (or XDP/kprobe sections). This:
- Avoids child process overhead
- Works in containerized/sandboxed environments
- Supports LD_IMM64 (64-bit immediate) decoding
- Prioritizes explicit BPF sections over `.text` fallback

## Installation Options

**Pre-built Binary**:
```bash
wget https://github.com/Nash0810/BPF-Insight/releases/latest/download/bpfva-linux-amd64
chmod +x bpfva-linux-amd64
sudo mv bpfva-linux-amd64 /usr/local/bin/bpfva
```

**From Source**:
```bash
make build      # Builds bpfva binary
make install    # Installs to /usr/local/bin/bpfva (requires sudo)
make test       # Runs test suite
```

## How It Works

### Analysis Pipeline

```
1. ELF PARSING
   └─ Extract bytecode from section (.text, xdp, kprobe, etc.)
   
2. INSTRUCTION DECODING
   └─ Convert raw bytes to structured instructions (opcode, operands)
   └─ Handle LD_IMM64 (64-bit immediates that span 2 instructions)
   
3. CFG CONSTRUCTION
   └─ Identify basic blocks (entry, jump targets, fallthrough)
   └─ Build edges (jump, fallthrough, back-edges)
   └─ Detect loops via DFS back-edge identification
   
4. REGISTER STATE SIMULATION
   └─ Initialize entry state (R1=context, R10=frame pointer)
   └─ Forward propagate states through blocks (worklist algorithm)
   └─ Merge states at block merges (join points)
   └─ Flag pointer arithmetic, R10 writes, unsafe memory access
   
5. COMPLEXITY SCORING
   └─ Calculate CFG metrics (depth, branching, loop count)
   └─ Sum rule violations with severity weights
   └─ Normalize to 0-100 scale
   
6. PREDICTION & RECOMMENDATIONS
   └─ Assign binary prediction (PASS/FAIL) with confidence
   └─ Identify top complexity hotspots
   └─ Suggest remediations based on detected patterns
```

### Complexity Score Interpretation

The score combines two factors:

**CFG Metrics (0-40 points)**
- Instruction count relative to 1M limit
- Control flow depth (deeper = more state combinations)
- Loop nesting and back-edges
- Branching factor per block
- Helper call count

**Rule Violations (0-60 points)**
- Critical violations = 15 pts each (e.g., unbounded loop)
- High = 10 pts each (pointer arithmetic)
- Medium = 5 pts each (suspicious shift)
- Low = 2 pts each (style issues)

**Score Ranges:**
- 0-24: Low risk (likely PASS)
- 25-49: Moderate risk (uncertain)
- 50-74: High risk (likely FAIL)
- 75-100: Critical risk (probable FAIL)

## Quick Command Reference

| Task | Command | Use Case |
|------|---------|----------|
| **Analyze single program** | `bpfva analyze prog.o` | Get complexity score and hotspots |
| **See detailed metrics** | `bpfva analyze prog.o --verbose` | Understand each metric component |
| **Generate CFG image** | `bpfva visualize prog.o --render` | Visualize control flow with heatmap |
| **Debug rule violations** | `bpfva verify prog.o --json` | Low-level rule engine output |
| **Compare optimization** | `bpfva compare before.o after.o` | Measure improvement |
| **Batch check programs** | `bpfva batch ./ebpf --recursive` | CI/CD integration |
| **Use strict rules** | `bpfva verify prog.o --profile strict` | Conservative analysis |
| **JSON output** | `bpfva analyze prog.o --json` | Integration with scripts |

## Limitations and Scope

This is a **heuristic-based reverse engineering project**, not a verifier emulator. Key limitations:

### Accuracy and Trade-offs

**What works well (~80% accuracy):**
- Binary predictions on instruction-heavy programs (>100 instructions)
- Detecting loops and branching complexity
- Catching pointer arithmetic violations
- Identifying R10 (frame pointer) writes

**Where it struggles:**
- Programs with complex verifier heuristics (state pruning, dead code elimination)
- Kernel version-dependent verifier behavior
- Helper function side effects on register states
- Complex state merges at loop boundaries

**Why not 100%:** The kernel verifier uses optimization techniques and undocumented heuristics that are difficult to reverse-engineer without access to verifier source code behavior on edge cases.

### Validation Methodology

Testing was conducted in a **local environment** (8GB RAM, x86_64, Ubuntu):
- 15 representative eBPF programs (simple XDP, kprobe, socket filters)
- Programs compiled with clang 14 targeting Linux 5.15 kernel
- Predictions verified by actually loading programs with `bpftool prog load`
- Accuracy calculated as: (correct predictions) / (total programs)

This is **not** a statistically rigorous benchmark across kernel versions or architectures.

### Known False Positives/Negatives

**False Positives** (predicts FAIL, actually PASS): ~10%
- Conservative register state tracking may flag safe operations
- Strict rules enabled by default

**False Negatives** (predicts PASS, actually FAIL): <5%
- Missed edge cases in state merging
- Kernel version specific verifier constraints

### What This Tool Is Not

- ❌ A verifier simulator (doesn't execute code)
- ❌ A semantic analyzer (doesn't verify correctness)
- ❌ Production-ready in isolation (use for early feedback, not final validation)
- ❌ Cross-version guaranteed (tested on one kernel config)

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

## Repository Structure

```
BPF-Insight/
├── cmd/                    # CLI commands
│   ├── main.go            # Entry point
│   ├── analyze.go         # Main analysis command
│   ├── verify.go          # Low-level rule engine
│   ├── visualize.go       # CFG rendering
│   ├── compare.go         # Differential analysis
│   └── batch.go           # Batch processing
│
├── pkg/
│   ├── parser/            # ELF & instruction decoding
│   ├── cfg/               # Control flow graph
│   ├── analyzer/          # Scoring & analysis
│   ├── verify/            # Rule engine & state tracking
│   └── utils/             # Output formatting
│
├── test/                  # Test programs
│   ├── programs/          # eBPF source (.c files)
│   ├── compiled/          # Pre-compiled .o files
│   └── validation/        # Validation test cases
│
├── docs/                  # Documentation
│   ├── METHODOLOGY.md     # Technical approach
│   ├── EXAMPLES.md        # Usage examples
│   └── ARCHITECTURE.md    # Design details
│
└── scripts/               # Build & test automation
```

## Development

### Local Build & Test
```bash
make build      # Compile bpfva
make test       # Run test suite
make validate   # Test against kernel verifier (requires sudo)
```

### Code Organization
For detailed architecture and design decisions, see [ARCHITECTURE.md](./ARCHITECTURE.md).

## Contributing

Contributions welcome. Before submitting:
- Run `make test` to verify changes
- Add tests for new functionality
- Reference relevant GitHub issues
- Follow existing code style

## What's Next?

**Learning Resources**:
- [METHODOLOGY.md](./docs/METHODOLOGY.md): Deep dive into verification constraints and CFG analysis
- [EXAMPLES.md](./docs/EXAMPLES.md): Practical usage scenarios
- [ARCHITECTURE.md](./ARCHITECTURE.md): Design decisions and implementation details

**Potential Improvements**:
- Multi-version kernel support (currently tested on 5.15)
- Parallel batch processing
- LSP server for IDE integration
- Extended dataflow analysis with field offset tracking

