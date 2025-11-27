# Usage Examples

## Use Case 1: Pre-Deployment Verification

**Scenario**: You've written an XDP program and want to check if it will pass the verifier before deploying.

```bash
# Compile your program
clang -O2 -target bpf -c xdp_firewall.c -o xdp_firewall.o

# Analyze it
bpfva analyze xdp_firewall.o

# Output shows high score
Complexity Score: 78.5 / 100
Prediction: LIKELY FAIL

# Get recommendations
bpfva analyze xdp_firewall.o --verbose

Recommendations:
  🔴 HIGH: Unbounded loop at Block 12
  🟠 MEDIUM: Excessive branching in Block 8
```

**Action**: Fix issues based on recommendations, recompile, re-analyze.

---

## Use Case 2: Optimizing Existing Programs

**Scenario**: Your program is rejected by the verifier. Use bpfva to identify issues.

```bash
# Analyze original
bpfva analyze original.o > original_report.txt

# Make optimizations based on recommendations
# ... edit source code ...

# Recompile
clang -O2 -target bpf -c optimized.c -o optimized.o

# Compare
bpfva compare original.o optimized.o

Output:
Complexity Scores:
  Before: 82.5 / 100 (LIKELY FAIL)
  After:  45.2 / 100 (MAY PASS)
  Delta:  -37.3 (significant improvement)

Metric Changes:
  Instructions: -234 (-22.8%)
  Loops: 3 → 1 (-66.7%)
```

---

## Use Case 3: CI/CD Integration

**Scenario**: Automatically check all eBPF programs in your repository.

```bash
# In your CI pipeline (e.g., GitHub Actions)

# Compile all programs
make compile-ebpf

# Batch analyze
bpfva batch ./ebpf/ --recursive --fail-threshold 70

# Exit code 1 if any program scores >70
# This blocks deployment of risky programs
```

**.github/workflows/ebpf-check.yml**:
```yaml
name: eBPF Verification Check

on: [push, pull_request]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install bpfva
        run: |
          wget https://github.com/Nash0810/BPF-Insight/releases/latest/download/bpfva-linux-amd64
          chmod +x bpfva-linux-amd64
          sudo mv bpfva-linux-amd64 /usr/local/bin/bpfva
      
      - name: Compile eBPF programs
        run: make compile-ebpf
      
      - name: Analyze programs
        run: bpfva batch ./ebpf/ --fail-threshold 70
```

---

## Use Case 4: Learning and Debugging

**Scenario**: Understand why a program is complex.

```bash
# Generate visualization
bpfva visualize complex_program.o --render

# Opens PNG showing:
# - Red blocks = high complexity hotspots
# - Dashed lines = loop back-edges
# - Labels show instruction ranges

# Get hotspot details
bpfva analyze complex_program.o --hotspots 10

Top Complexity Hotspots:
  1. Block 15 (insns 345-389) - Score: 52.0
     └─ Loop header with deep nesting
  2. Block 8 (insns 156-234) - Score: 48.5
     └─ Excessive branching (5 successors)
```

---

## Use Case 5: Multi-Version Testing

**Scenario**: Ensure your program works across kernel versions.

```bash
# Test against different kernel verifier configs
# (using Docker containers with different kernels)

for kernel_version in 5.10 5.15 6.1; do
    docker run --rm -v $(pwd):/work kernel:$kernel_version \
        bpfva analyze /work/program.o --json \
        > results_$kernel_version.json
done

# Compare results
jq '.score' results_*.json
```

---

## Use Case 6: Code Review Helper

**Scenario**: Review teammate's eBPF program PR.

```bash
# Analyze their program
bpfva analyze new_feature.o --verbose > review.txt

# Add review.txt to PR comments showing:
# - Complexity score
# - Identified issues
# - Specific line numbers to check

# Example review comment:
"""
Analysis shows complexity score of 65.5 (medium risk).

Issues found:
1. Unbounded loop at insn 234 - consider adding explicit bound
2. Missing bounds check before packet access at insn 456

Please address before merging.
"""
```

