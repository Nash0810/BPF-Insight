# Technical Methodology

## The Kernel eBPF Verifier: A Brief Overview

The Linux kernel verifier is a static analyzer that validates eBPF programs before allowing them to run. It enforces four main categories of constraints:

### 1. **Instruction Processing Limit** (1M instructions/program)

Each branch creates a new verification path. The verifier explores paths up to a limit:
```
Instructions processed = Σ(instructions per path) across all reachable paths
```

For example:
```
if (x < 10) {
    do_work();  // Path A: 50 instructions
} else {
    do_other(); // Path B: 60 instructions
}
// Verifier processes ~110 instructions (assuming linear paths)
```

But with nested branches:
```
if (...) {      // 2 paths
    if (...) {  // 2 × 2 = 4 paths
        if (...) {  // 2 × 2 × 2 = 8 paths
```

The verifier explores all 2^depth paths, causing exponential growth. This is why complexity often comes from **depth** rather than raw instruction count.

### 2. **State Explosion** (Register Value Range Tracking)

The verifier tracks value ranges for each register:
```
R1 = [0, 4096]           // Pointer to packet buffer, length 4096
R2 = [0, 255]            // Filter value
R3 = unknown             // Uninitialized
```

At each branch, state is duplicated:
```
State after 10 instructions: 1 state
State after 10 branches: up to 2^10 = 1024 states
```

The verifier attempts to **prune** redundant states, but complex arithmetic defeats this.

### 3. **Loop Unrolling** (Bounded Loop Requirement)

The verifier cannot handle unbounded loops. It requires explicit bounds:

**Safe (bounded)**:
```c
#pragma unroll
for (int i = 0; i < 100; i++) {
    process_packet_byte();
}
```

**Unsafe (unbounded)**:
```c
while (data < end) {        // Verifier can't prove loop termination
    data++;
}
```

### 4. **Pointer Arithmetic Validation**

The verifier tracks pointer provenance (origin) and bounds:

**Safe**:
```c
entry = bpf_map_lookup_elem(&map, &key);  // Verifier knows map value size
offset = bpf_ntohl(data);                  // Convert to known range
access_ptr = entry + offset % 64;          // Bounds checked
```

**Unsafe**:
```c
ptr = get_ptr_somehow();          // Origin unclear
offset = user_data * 100;          // Unbounded
access = ptr + offset;             // May crash
```

---

## BPF-Insight's Reverse Engineering Approach

We approximate verifier behavior through **three layers** of analysis:

### Layer 1: Control Flow Structure

**Goal**: Understand how many execution paths exist

**Method**: Build a Control Flow Graph (CFG)
- Each basic block has exactly one entry and one exit
- Entry = program start or jump target
- Exit = jump or return instruction
- Edges represent possible transitions

**Why it matters**: The CFG depth correlates with state explosion. Deep paths = more states.

**Algorithm** (leader-based block identification):
```
1. Find "leaders" (block entry points):
   - Program entry (instruction 0)
   - Jump targets
   - Instruction after each jump
   
2. Partition instructions into blocks:
   - Block starts at leader
   - Block ends at jump or return
   
3. Connect blocks:
   - Jump instruction → edge to target
   - Fallthrough → edge to next block
   - Return → no outgoing edges
```

**Example**:
```
0:  mov r0, 1
1:  if r1 < 100 goto 4      ← Jump target (leader)
2:  mov r0, 0                ← Fallthrough leader
3:  exit
4:  mov r0, 2                ← Jump target (leader)
5:  exit

CFG:
Block 0: [0, 1]  ├─→ Block 1: [2, 3]
                 └─→ Block 2: [4, 5]
```

### Layer 2: Register State Simulation

**Goal**: Track which registers hold pointers vs. constants

**Method**: Abstract interpretation
- Assign each register a type: UNKNOWN, CONST(value), POINTER, STACK_PTR, MAP_PTR
- At each instruction, update types based on what the instruction does
- When encountering pointer arithmetic, check if it's valid

**Register Type Lattice**:
```
UNKNOWN (top — most permissive)
  ├─ CONST(value)      ← Known constant
  ├─ POINTER           ← Some pointer type
  │   ├─ MAP_PTR       ← Pointer from bpf_map_lookup_elem
  │   ├─ STACK_PTR     ← Pointer to stack
  │   └─ CTX_PTR       ← Context pointer (R1)
  └─ TAINTED           ← User input, could be anything
```

**State Propagation Algorithm**:
```
Initial state: R1 = CTX_PTR, R10 = STACK_PTR, others = UNKNOWN

For each instruction:
  If R2 = R1 + 4:
    R2 = POINTER (copy type)
  
  If R2 = R2 + R3 (where R3 is unknown):
    ERROR: Pointer arithmetic with unknown offset
    R2 = UNKNOWN (invalidate)
  
  If R2 = bpf_map_lookup_elem(...):
    R2 = MAP_PTR
  
  If R2 = R2 * 100:
    ERROR: Arithmetic on pointer
    R2 = UNKNOWN
```

**Block Merge**: When two paths converge, combine their register states:
```
Path A: R1 = CONST(5)
Path B: R1 = CONST(10)
Merged: R1 = UNKNOWN (because value could be either)

Path A: R2 = MAP_PTR
Path B: R2 = POINTER
Merged: R2 = POINTER (conservative: may not be MAP_PTR)
```

### Layer 3: Complexity Scoring

**Goal**: Convert multiple metrics into a single 0-100 score

**Metrics Captured**:

1. **Instruction count** (weight: 40 pts)
   - Relative to 1M limit: (count / 1M) × 40
   - Example: 100k instructions = 4 points

2. **Control flow depth** (weight: 25 pts)
   - Longest path from entry: (max_depth / 100) × 25
   - Example: 50-block deep = 12.5 points

3. **Loop complexity** (weight: 10 pts)
   - Each back-edge = 1 loop: back_edges × 10
   - Example: 3 loops = 30 points (but capped at total)

4. **Branching factor** (weight: 5 pts)
   - Average successors per block: (avg_branch / 10) × 5
   - Example: 2.5 avg successors = 1.25 points

5. **Helper calls** (weight: 5 pts)
   - Each helper may modify state: (calls / 50) × 5
   - Example: 10 helpers = 1 point

**Final Score**:
```
raw_score = insn_score + depth_score + loop_score + branch_score + helper_score
rule_penalties = sum of all rule violation penalties
final_score = min(100, raw_score + rule_penalties)
```

**Rule Penalties**:
- Critical violation (unbounded loop, write to R10): +15 points
- High violation (pointer arithmetic): +10 points
- Medium violation (suspicious shifts): +5 points
- Low violation (style): +2 points

---

## How This Translates to Predictions

### Confidence Levels

BPF-Insight assigns confidence based on analysis characteristics:

**WILL_FAIL** (Confidence: 95%+)
- Score ≥ 75
- Multiple critical violations detected
- Example: Unbounded loop + pointer arithmetic

**LIKELY_FAIL** (Confidence: 70-95%)
- Score 50-74
- Several high-severity issues
- Example: Deep nesting + no bounds checking

**MAY_PASS** (Confidence: Uncertain)
- Score 25-49
- Some patterns detected but not conclusive
- Might pass on specific kernel version

**LIKELY_PASS** (Confidence: 70-95%)
- Score 5-24
- Clean structure, few patterns
- Most kernel versions accept

**WILL_PASS** (Confidence: 95%+)
- Score < 5
- Trivial program, no violations
- Example: Simple packet counter

### Why We Can't Reach 100% Accuracy

The kernel verifier uses **optimizations** we can't fully reverse-engineer:

1. **State Pruning**: Merges states that will behave identically
   - We can detect when pruning *might* happen
   - Can't predict exactly when verifier prunes

2. **Dead Code Elimination**: Recognizes unreachable paths
   - Reduces effective path count
   - We count all paths conservatively

3. **Value Range Narrowing**: Tightens bounds based on code patterns
   - Example: `if (x < 100) { x + 1; }` → Verifier knows x ≤ 100
   - Our abstract interpretation may be more conservative

4. **Helper Function Specifics**: Each helper has side effects
   - `bpf_get_current_pid_tgid()` is deterministic
   - `bpf_skb_load_bytes()` affects state complexity
   - We can't perfectly model all 180+ helpers

5. **Kernel Version Drift**: Verifier changes between versions
   - 5.10, 5.15, 6.1, 6.6+ have different limits
   - We test against one kernel version

