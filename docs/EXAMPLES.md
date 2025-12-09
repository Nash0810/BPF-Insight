# Usage Examples

## Example 1: Diagnosing a Verifier Rejection

**Scenario**: Your XDP program was rejected by the verifier with cryptic error messages. Use bpfva to identify root causes.

```bash
# Compile with optimization disabled for easier debugging
clang -O2 -target bpf -c xdp_firewall.c -o xdp_firewall.o

# Analyze with full details
$ bpfva analyze xdp_firewall.o --verbose --hotspots 10

eBPF Verifier Complexity Analysis
==================================
File: xdp_firewall.o
Section: xdp

Metrics:
  Instructions:     487
  Basic Blocks:     28
  Max Depth:        12
  Loops Detected:   2
  Avg Branching:    1.8
  Helper Calls:     5

Complexity Score: 58.3 / 100
Prediction: LIKELY_FAIL

Top Complexity Hotspots:
  1. Block 15 (insns 245-289) - Score: 48.5
     └─ Loop header with 7 instructions and 2 successors
  2. Block 18 (insns 301-334) - Score: 42.1
     └─ Pointer arithmetic on inferred register
  3. Block 8 (insns 128-167) - Score: 35.7
     └─ Excessive branching (4 successors)

Recommendations:
  🔴 CRITICAL: Unbounded loop at Block 15
     └─ Loop condition at insn 256 doesn't restrict iteration count
     └─ Consider: Add explicit iteration limit (#pragma unroll)
  
  🔴 HIGH: Pointer arithmetic at Block 18, insn 312
     └─ R3 = R2 + R4, where R4 has unknown bounds
     └─ Consider: Validate offset before use
  
  🟠 MEDIUM: Suspicious shift amount at insn 189
     └─ SHL R1, 16 (shift by 16 bits on register that may hold pointer)
     └─ Consider: Use bpf_ntohs() instead of manual bit ops
```

**Analysis**: The score of 58.3 indicates medium-to-high risk. The unbounded loop is the main issue—the verifier can't prove it terminates. The pointer arithmetic makes things worse.

**Action**: Apply recommendations:
```c
// Before
while (ptr < end) {        // No bound → unbounded loop
    process(ptr);
    ptr += offset;          // Offset not validated
}

// After
#pragma unroll
for (int i = 0; i < 100; i++) {  // Explicit bound
    if (ptr >= end) break;
    process(ptr);
    validated_offset = offset & 0xFF;  // Mask to known range
    ptr += validated_offset;
}
```

Recompile and re-analyze:
```bash
$ bpfva analyze xdp_firewall_fixed.o
Complexity Score: 32.1 / 100
Prediction: LIKELY_PASS
```

**Result**: Score dropped 26 points. Program now likely acceptable.

---

## Example 2: Measuring Optimization Impact

**Scenario**: You've refactored your packet processing logic. Did you actually improve verifier acceptance?

```bash
# Analyze original version
$ bpfva analyze original_version.o --json > before.json

# Analyze refactored version
$ bpfva analyze refactored_version.o --json > after.json

# Compare
$ bpfva compare original_version.o refactored_version.o

Comparison Report
=================

Complexity Scores:
  Before: 68.4 / 100 (LIKELY_FAIL)
  After:  35.2 / 100 (LIKELY_PASS)
  Delta:  -33.2 (43% improvement) ✓

Metric Changes:
  Instructions:    456 → 312 (-68, -14.9%)
  Basic Blocks:    24 → 18 (-6)
  Max Depth:       15 → 8 (-7, -46.7%)
  Loops Detected:  3 → 1 (-2, -66.7%)
  Avg Branching:   2.1 → 1.4 (-0.7, -33%)
  Helper Calls:    7 → 4 (-3)

Violations Resolved:
  ✓ Unbounded loop at Block 12 (now bounded with #pragma)
  ✓ Excessive branching in Block 8 (refactored to lookup table)

Violations Introduced:
  ✗ Suspicious shift at insn 256 (unlikely to cause rejection)
```

**Interpretation**: The refactoring reduced complexity by ~43%. The key improvements:
- **Loop bounding** (3 loops → 1): Each loop the verifier must unroll costs instructions
- **Depth reduction** (15 → 8): Shallower paths = fewer state combinations
- **Branching reduction** (2.1 → 1.4 avg): Fewer branches = linear paths rather than exponential

The new suspicious shift is a minor issue unlikely to cause rejection on modern kernels.

---

## Example 3: CI/CD Integration and Gating

**Scenario**: Your repo contains multiple eBPF programs. You want CI to reject PRs that introduce high-risk programs.

```bash
# In Makefile
.PHONY: ebpf-check
ebpf-check:
	@echo "Checking eBPF program complexity..."
	bpfva batch ./ebpf/ --recursive --output-format json > .ebpf-check.json
	@# Exit with non-zero if any program scores > 70
	@jq 'if any(.results[]; .score > 70) then 1 else 0 end' .ebpf-check.json
```

**.github/workflows/ci.yml**:
```yaml
name: eBPF Verification Gate

on: [push, pull_request]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install bpfva
        run: |
          wget -q https://github.com/Nash0810/BPF-Insight/releases/latest/download/bpfva-linux-amd64
          chmod +x bpfva-linux-amd64
          sudo mv bpfva-linux-amd64 /usr/local/bin/bpfva
      
      - name: Compile eBPF programs
        run: make compile-ebpf
      
      - name: Check complexity
        run: |
          bpfva batch ./ebpf/ --recursive --fail-threshold 70 --output results.txt
          cat results.txt
      
      # Optional: Post comment to PR with analysis
      - name: Comment on PR
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const results = fs.readFileSync('results.txt', 'utf-8');
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '## eBPF Complexity Analysis\n```\n' + results + '\n```'
            });
```

**Output**:
```
Batch Analysis Report
=====================
Directory: ./ebpf/
Files found: 8
Analyzed: 8
Errors: 0

Results Summary:
├─ packet_filter.o: 24.5 (LIKELY_PASS) ✓
├─ rate_limiter.o: 45.3 (MAY_PASS) ⚠
├─ packet_processor.o: 72.1 (LIKELY_FAIL) ✗
├─ debug_helper.o: 18.2 (LIKELY_PASS) ✓
├─ classifier.o: 63.4 (HIGH_RISK) ✗
├─ counter.o: 12.0 (LIKELY_PASS) ✓
├─ logger.o: 31.2 (LIKELY_PASS) ✓
└─ monitor.o: 68.9 (HIGH_RISK) ✗

High-Risk Programs: 3
  - packet_processor.o (score: 72.1)
  - classifier.o (score: 63.4)
  - monitor.o (score: 68.9)

Average Complexity: 41.8
Median Complexity: 37.8

Exit Code: 1 (threshold 70 exceeded)
```

**Action**: The CI gate blocks the PR and posts a comment. The PR author sees three high-risk programs and addresses them before resubmitting.

---

## Example 5: Deep-Dive Rule Engine Debugging

**Scenario**: You want to understand exactly which rules are triggering and why. Use the low-level `verify` command.

```bash
# Run rule engine with JSON output
$ bpfva verify complex_program.o --json

{
  "BlockWarnings": [
    {
      "BlockID": 15,
      "Message": "Pointer arithmetic on inferred register: R2 = R1 + R3"
    },
    {
      "BlockID": 18,
      "Message": "Shift amount 16 on register holding possible pointer"
    }
  ],
  "ProgramWarnings": [
    "Unbounded loop detected at Block 12: no iteration counter found",
    "R10 (frame pointer) write detected at Block 8",
    "Map lookup at Block 5 without null check at Block 6"
  ],
  "FinalPrediction": "HIGH RISK"
}
```

**Rule-level debugging with profiles**:
```bash
# Strict checking (all rules enabled)
$ bpfva verify program.o --profile strict
[Program] Unbounded loop detected at Block 12
[Program] R10 write detected at Block 8
[Program] Map lookup without null check

# Permissive checking (only critical rules)
$ bpfva verify program.o --profile permissive
[Program] Unbounded loop detected at Block 12

# Selective rule enable/disable
$ bpfva verify program.o --disable pointer_arithmetic --enable map_no_null_check
[Program] Map lookup without null check
```

**Understanding specific rules**:

If you're curious about why a rule triggers:
```bash
# Get verbose recommendations
$ bpfva analyze program.o --verbose

Recommendations:
  🔴 CRITICAL: Unbounded loop (pattern: unbounded_loop)
     Location: Block 12, insn 245-256
     Issue: Loop condition at insn 251 doesn't restrict iteration count
     └─ Suggestion: Add explicit iteration limit using #pragma unroll

  🔴 HIGH: Pointer arithmetic (pattern: pointer_arithmetic)
     Location: Block 18, insn 312
     Issue: R3 = R2 + R4, where R4 has unknown bounds
     └─ Suggestion: Validate offset before arithmetic: R4 &= 0xFF
```

This maps directly to the low-level rules detected by `verify`.

## Example 4: Visual Complexity Analysis with CFG Rendering

**Scenario**: You want to understand the control flow structure visually and identify bottlenecks.

```bash
# Generate CFG visualization with complexity heatmap
$ bpfva visualize packet_processor.o --render --format png

Generating CFG visualization...
  └─ Parsed 487 instructions
  └─ Built CFG with 28 basic blocks
  └─ Detected 2 loops
  └─ Rendering to PNG...
  └─ Saved: packet_processor.o.png

# View the image (opens in default viewer or you can view manually)
```

**What the visualization shows**:
- **Node colors**:
  - 🟢 Green blocks: Low complexity hotspots (score < 20)
  - 🟡 Yellow blocks: Medium complexity (score 20-40)
  - 🟠 Orange blocks: High complexity (score 40-60)
  - 🔴 Red blocks: Critical hotspots (score > 60)

- **Edge styles**:
  - Solid lines: Fallthrough edges (sequential execution)
  - Dashed lines: Back-edges (loop targets)
  - Arrows: Jump targets

- **Node labels**: 
  ```
  [Block 15]
  insn 245-289 (9 instructions)
  Successors: 2
  Score: 48.5
  ```

**Interpretation**: If most blocks are green/yellow, the program is straightforward. Red clusters indicate problematic areas.

**Example output analysis**:
```
CFG structure identified:
- Entry block (small, green)
- Main loop (red cluster) ← Focus optimization here
  - 7 blocks in loop body
  - Deepest nesting point
- Error handling (yellow) ← Secondary optimization target
- Exit blocks (green)
```

