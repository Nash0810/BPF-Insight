# BPF-Insight Architecture

## Design Principles

**Modularity**: Each phase (parsing, CFG, simulation, rules) is independent and testable  
**Single Responsibility**: Packages have one clear job (parser parses, verifier verifies, etc.)  
**Conservative Analysis**: When unsure, assume the worst (e.g., unknown values = tainted)  
**No External Dependencies**: Avoid shell calls; parse ELF directly  

---

## Package Organization

### `cmd/` — Command Line Interface

Five independent subcommands using Cobra framework:

**analyze.go**: Main entry point
- Orchestrates full pipeline: ELF parsing → CFG → analysis → scoring
- Supports flags for JSON output, verbosity, CFG visualization
- Produces `ComplexityReport` with hotspots and recommendations

**verify.go**: Debugger-focused rule engine output
- Bypasses complexity scoring; shows raw rule violations
- Useful for understanding *why* patterns trigger
- Supports rule enable/disable and profiles

**visualize.go**: CFG rendering
- Generates DOT format with color heatmaps
- Optionally renders to PNG/SVG via Graphviz
- Hotspots colored by complexity score (red = high, green = low)

**compare.go**: Differential analysis
- Analyzes before/after versions
- Shows score delta, metric changes, resolved/new violations
- Useful for measuring optimization impact

**batch.go**: Bulk processing
- Recursively finds all `.o` files in directory
- Produces aggregate report (avg score, high-risk programs)
- Returns exit code based on threshold for CI/CD

---

### `pkg/parser/` — ELF Parsing and Instruction Decoding

**elf.go**: Custom ELF parser (zero external dependencies)
- Opens ELF file directly with Go's `debug/elf` library
- Prioritizes named sections (xdp, kprobe, etc.) over `.text` fallback
- Extracts raw bytecode from section data

**decode.go**: eBPF instruction decoding
- Converts raw bytes to structured `parser.Instruction` type
- Handles LD_IMM64 (64-bit immediate instructions that span 2 bytes)
- Extracts opcode, operands, offsets for later analysis

**instruction.go**: Instruction metadata
- Defines `Instruction` struct with semantic info
- Methods: `IsJump()`, `IsExit()`, `GetHelper()` for fast pattern matching
- Supports register and operand extraction

**convert.go**: AST conversion
- Bridges `parser.Instruction` to `cilium/ebpf.asm.Instruction`
- Enables compatibility with external eBPF libraries if needed

---

### `pkg/cfg/` — Control Flow Graph Analysis

**cfg.go**: CFG construction
- `BuildBasicBlocks()`: Identifies block leaders using classic algorithm
  - Leaders: program entry, jump targets, post-jump fallthrough
  - Partitions instructions into blocks
- `BuildCFG()`: Connects blocks with edges
  - Identifies edge types: fallthrough, jump, back-edge
  - Stores predecessor/successor relationships

**score.go**: Complexity metrics
- `CalculateScores()`: Computes per-block and program-wide metrics
- Metrics:
  - Max depth (longest path from entry to any block)
  - Average branching (total successors / block count)
  - Loop count (number of back-edges)
  - Per-block scores based on depth, instruction count, successors

**dot.go**: Visualization generation
- Converts CFG to DOT format (Graphviz)
- Colors nodes by complexity score
- Labels show instruction ranges and hotspot status
- Can be rendered to PNG/SVG with `dot` command

---

### `pkg/verify/` — Verifier Rule Engine

**verify.go**: Main orchestrator
- `VerifyProgram()`: Runs full verification pipeline
- Performs two passes:
  1. **Dataflow analysis**: Forward propagate register states through blocks
  2. **Rule checking**: Apply rules at block/program level
- Uses worklist algorithm for state merging at block joins
- Combines block and program warnings into `VerifyOutput`

**regstate.go**: Register state tracking
- `RegState` struct tracks all 11 registers (R0-R10)
- Each register has type info: CONST, POINTER, STACK_PTR, etc.
- Methods:
  - `Apply(instruction)`: Updates state based on instruction effects
  - `Merge(other)`: Combines two states at block merge (conservative join)
  - `Clone()`: Deep copy for successor block initialization

**rules.go**: Rule implementation
- Global `Rules` map registers rule definitions
- Each rule has:
  - `BlockCheck(block, insn)`: Per-instruction checking
  - `BlockCheckState(block, insn, regstate)`: Stateful checking (pointer arithmetic, etc.)
  - `ProgramCheck(blocks)`: Whole-program analysis (map safety, helper chains)
- Rules are registered at init time

**ruleregistry.go**: Rule registry
- `Rule` struct definition
- Rule enable/disable logic
- Rule instantiation (happens at package init)

**profiles.go**: Verification profiles
- Preset rule configurations: "strict", "default", "permissive"
- `ApplyProfile()` enables/disables rules based on profile name
- Allows different strictness levels for different use cases

**helpers.go, aliases.go**: Utility functions
- Helper function name lookups (e.g., which helpers exist)
- Register aliases and ABI conventions

---

### `pkg/analyzer/` — Complexity Analysis and Reporting

**complexity.go**: Main analysis orchestrator
- `Analyze()`: Combines all analysis phases
  1. Calculate CFG metrics
  2. Run rule engine
  3. Compute hotspots
  4. Calculate final score
- Produces `ComplexityReport` with score, prediction, recommendations

**comparator.go**: Differential analysis
- `ComparePrograms()`: Analyzes before/after versions
- Calculates score deltas and metric changes
- Shows which violations were resolved/added

---

### `pkg/utils/` — Output Utilities

**json.go**: JSON marshaling
- Serializes reports to JSON
- Pretty-prints for readability
- Excludes internal fields (like CFG) from JSON

---

## Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│ User Input: program.o file                                  │
└──────────────┬──────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────┐
│ pkg/parser/elf.go: ELF Parsing                              │
│ • Open ELF file                                             │
│ • Find code section (.text, xdp, kprobe, etc.)             │
│ • Extract raw bytecode                                      │
└──────────────┬──────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────┐
│ pkg/parser/decode.go: Instruction Decoding                  │
│ • Convert bytes → Instruction structs                       │
│ • Handle LD_IMM64 (2-instruction immediates)                │
│ • Extract opcode, operands, offsets                         │
└──────────────┬──────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────┐
│ pkg/cfg/cfg.go: CFG Construction                            │
│ • Identify block leaders (entry, jump targets)              │
│ • Partition into basic blocks                               │
│ • Connect with edges (jump, fallthrough, back-edge)         │
│ • Detect loops via DFS                                      │
└──────────────┬──────────────────────────────────────────────┘
               │
           ┌───┴────────────────────────────────────────┐
               │                                        │
               ▼                                        ▼
┌──────────────────────────────┐    ┌──────────────────────────────┐
│ pkg/cfg/score.go:            │    │ pkg/verify/verify.go:        │
│ Complexity Scoring           │    │ Rule Engine                  │
│ • Max depth calculation      │    │ • Dataflow analysis          │
│ • Avg branching             │    │ • Register state tracking    │
│ • Per-block hotspots        │    │ • Rule application           │
│ • Instruction metrics       │    │ • Violation collection       │
└──────────────┬───────────────┘    └──────────────┬───────────────┘
               │                                    │
               └────────────────┬───────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│ pkg/analyzer/complexity.go: Analysis Orchestration          │
│ • Combine CFG metrics and rule penalties                    │
│ • Calculate final score (0-100)                             │
│ • Generate hotspots list                                    │
│ • Make prediction (PASS/FAIL)                               │
│ • Generate recommendations                                  │
└──────────────┬──────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────┐
│ ComplexityReport                                            │
│ • Score, prediction, hotspots                               │
│ • Recommendations with severity                             │
│ • Basic metrics                                             │
└──────────────┬──────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────┐
│ Output (cmd/)                                               │
│ • Text formatting (analyze.go)                              │
│ • JSON serialization (utils/json.go)                        │
│ • DOT visualization (cfg/dot.go)                            │
└─────────────────────────────────────────────────────────────┘
```

---

## Key Algorithms

### Basic Block Identification (Leader-Based)

```
Input: []Instruction
Output: []*BasicBlock

leaders = {0}  // Entry point always a leader

// Identify leaders
for i, insn in instructions:
    if isJump(insn):
        target = calculateTarget(i, insn)
        leaders.add(target)        // Jump target is leader
        leaders.add(i + 1)         // Fallthrough is leader

// Partition into blocks
blocks = []
currentBlock = nil
for i, insn in instructions:
    if i in leaders and currentBlock is not empty:
        blocks.append(currentBlock)
        currentBlock = newBlock()
    currentBlock.add(insn)

if currentBlock is not empty:
    blocks.append(currentBlock)
```

**Complexity**: O(N) where N = instruction count

### Register State Propagation (Worklist)

```
Input: []BasicBlock, entry block
Output: Map[BlockID] → RegState

entryState = {entry_block.id: initialRegState()}

worklist = [entry_block.id]
visited = {}

while worklist not empty:
    bid = worklist.pop()
    block = blocks[bid]
    state = entryState[bid]
    
    // Simulate instructions in block
    for insn in block.instructions:
        for rule in rules:
            if rule.hasStateCheck:
                rule.check(block, insn, state)
        state.apply(insn)  // Update register state
    
    // Propagate to successors
    for succ in block.successors:
        if succ.id not in entryState:
            entryState[succ.id] = state.clone()
            worklist.push(succ.id)
        else:
            // Merge states
            if entryState[succ.id].merge(state):
                worklist.push(succ.id)  // Changed, re-process
```

**Complexity**: O(B × N × R) where:
- B = number of blocks
- N = avg instructions per block
- R = rules to check

Worst case: O(B²) iterations (loopy graphs)

### Loop Detection (DFS Back-Edge)

```
Input: CFG with entry block
Output: []LoopInfo (list of loops with headers and blocks)

visited = {}
rec_stack = {}  // Recursion stack (for back-edge detection)
back_edges = []

dfs(block):
    visited[block.id] = true
    rec_stack[block.id] = true
    
    for succ in block.successors:
        if succ.id not in visited:
            dfs(succ)
        elif rec_stack[succ.id]:
            // Back-edge: succ is ancestor
            back_edges.append((block, succ))
    
    rec_stack[block.id] = false

// Extract loops from back-edges
loops = extractLoops(back_edges)
```

**Complexity**: O(B + E) where B = blocks, E = edges

---

## Testing Strategy

### Unit Testing
- Parser: Decode specific instruction sequences, verify opcode extraction
- CFG: Known programs → expected block/edge structure
- Register state: Instruction sequences → expected state changes
- Rules: Programs with known violations → expected rule triggers

### Integration Testing
- Full pipeline: Compile program → analyze → verify output format
- Batch processing: Multiple files → aggregate report
- Visualization: CFG → DOT → manual inspection

### Validation Testing (Against Real Verifier)
- Compile representative programs
- Load with `bpftool prog load` (requires root)
- Compare predictions to actual accept/reject
- Track accuracy metrics

---

## Extension Points

**Adding a New Rule**:
1. Implement check function: `func newRuleCheck(block, insn, state) []string`
2. Register in `pkg/verify/rules.go`: `Rules["rule_name"] = &Rule{...}`
3. Define severity and description
4. Add to profile(s) if needed

**Adding New Output Format**:
1. Implement formatter: `func Format(report *ComplexityReport) string`
2. Register in command handler
3. Add CLI flag support

**Extending Register State**:
1. Add field to `RegState` struct
2. Implement `Apply()` logic for new field
3. Update `Merge()` for state joining
4. Extend rules that depend on this state

---

## Performance Characteristics

All measurements from local testing (8GB RAM, x86_64):

| Phase | Time | Notes |
|-------|------|-------|
| ELF Parse | <1ms | Fast file I/O |
| Decode Instructions | ~0.1ms per 100 insns | Linear with size |
| CFG Construction | ~0.2ms per 100 insns | Leader identification + edge building |
| Register Simulation | ~1-5ms per 100 insns | Depends on block complexity |
| Rule Checking | ~0.5ms per 100 insns | Depends on rules enabled |
| Scoring | <0.1ms | Simple arithmetic |
| **Total** | **~2-10ms** | For typical 100-500 insn program |

Batch processing (100 files): ~0.5-1.0s total (parallelizable if needed)

---

## Future Improvements

1. **Stateful Rule Refinement**: Current pointer tracking is conservative; could improve precision with context-sensitive analysis
2. **Kernel Version Profiles**: Different rules/thresholds for different kernel versions
3. **Parallel Batch Processing**: Currently sequential; could spawn workers for each file
4. **Machine Learning Integration**: Train model on true verifier behavior to improve accuracy beyond heuristics
5. **LSP Server**: IDE integration for real-time analysis during development
