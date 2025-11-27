# Technical Methodology

This document explains the technical approach used by bpfva.

## eBPF Verifier Complexity

### What the Verifier Does

The Linux kernel's eBPF verifier performs static analysis to ensure programs:
1. Will terminate (no infinite loops)
2. Won't access invalid memory
3. Won't crash the kernel
4. Respect security boundaries

### Complexity Sources

**1. Instruction Processing Limit**
- The verifier processes up to 1,000,000 instructions
- Each branch creates new verification paths
- Complex programs exhaust this limit

**2. State Explosion**
- Each register has a tracked value range
- Each branch duplicates state
- Formula: states ≈ branches × depth × register_combinations

**3. Loop Unrolling**
- Loops must have provable bounds
- Verifier unrolls loops up to limit
- Unbounded loops cause rejection

**4. Pointer Tracking**
- Pointer arithmetic must be verifiable
- Complex calculations confuse tracking
- Out-of-bounds access rejected

## Our Approach

### Control Flow Graph Construction

```
Step 1: Identify Basic Blocks
  - Start: program entry, jump targets
  - End: jump instructions, program exit

Step 2: Build Edges
  - Unconditional jump → single edge
  - Conditional jump → two edges (taken/not-taken)
  - Fallthrough → edge to next block

Step 3: Detect Loops
  - Run DFS from entry
  - Back-edges = edges to ancestors
  - Back-edge count = loop complexity
```

### Complexity Scoring

**Formula:**
```
score = min(100,
    (instructions / 1M) × 40 +
    (max_depth / 100) × 25 +
    (back_edges × 10) +
    (avg_branching × 5) +
    (helper_calls / 50) × 5
)
```

**Rationale:**
- Instruction count: Primary verifier limit
- Depth: Deep paths multiply states
- Back-edges: Loops cause unrolling
- Branching: More branches = more states
- Helpers: Modify register states

### Pattern Detection

Each pattern has:
- **Detection heuristic**: How we identify it
- **Severity**: Impact on verifier
- **Recommendation**: How to fix

**Example: Unbounded Loop Detection**
```go
// Pseudocode
for each back_edge:
    loop_block = back_edge.target
    
    has_counter = false
    for insn in loop_block:
        if insn is_comparison_with_constant:
            has_counter = true
    
    if not has_counter:
        report_unbounded_loop()
```

## Validation Methodology

We validate predictions by:
1. Compiling test programs with known characteristics
2. Running tool to get predictions
3. Loading programs with bpftool (requires root)
4. Comparing predictions to actual outcomes

**Accuracy Calculation:**
```
accuracy = (correct_predictions / total_programs) × 100
```

Where correct = (predicted PASS and actual PASS) OR (predicted FAIL and actual FAIL)

## Limitations

### Why 100% Accuracy is Impossible

1. **Verifier Heuristics**: The real verifier uses undocumented heuristics
2. **Kernel Version Differences**: Verifier behavior changes across versions
3. **State Pruning**: Complex optimization in real verifier
4. **Helper Function Effects**: Helper behavior affects verification

### Our Target

We aim for:
- >80% accuracy on binary predictions (PASS/FAIL)
- >90% accuracy including MAY_PASS uncertainty
- <5% false negatives (predicting PASS when actually FAIL)

## References

- Linux kernel source: `kernel/bpf/verifier.c`
- "eBPF Verifier Deep Dive" - Brendan Gregg
- BPF and XDP Reference Guide - Cilium

