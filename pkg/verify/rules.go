package verify

import (
	"fmt"

	"github.com/Nash0810/BPF-Insight/pkg/cfg"
	"github.com/Nash0810/BPF-Insight/pkg/parser"
)

// Register rules on init
func init() {

	// 1. Pointer arithmetic
	Rules["pointer_arithmetic"] = &Rule{
		Name:        "pointer_arithmetic",
		Description: "Detect ALU arithmetic on pointer registers",
		Enabled:     true,
		BlockCheck:  rulePointerArithmetic,
	}

	// 2. Helper chain
	Rules["helper_chain"] = &Rule{
		Name:         "helper_chain",
		Description:  "Detect multiple helper calls inside a single block",
		Enabled:      true,
		ProgramCheck: ruleHelperChain,
	}

	// 3. Stack variable offset
	Rules["stack_var_offset"] = &Rule{
		Name:        "stack_var_offset",
		Description: "Detect stack accesses with unverifiable offsets",
		Enabled:     true,
		BlockCheck:  ruleStackVarOffset,
	}

	// 4. Write to R10
	// Converted to program-level to increase severity impact
	Rules["write_r10"] = &Rule{
		Name:         "write_r10",
		Description:  "Detect writes to R10 (frame pointer)",
		Enabled:      true,
		ProgramCheck: ruleWriteR10Program,
	}

	// 5. Map lookup without null-check
	Rules["map_no_null_check"] = &Rule{
		Name:         "map_no_null_check",
		Description:  "Detect map lookup result used without null check",
		Enabled:      true,
		ProgramCheck: ruleMapLookupNoNull,
	}

	// 6. Map update without key validation
	Rules["map_update_no_key_check"] = &Rule{
		Name:         "map_update_no_key_check",
		Description:  "Detect map update without validating key",
		Enabled:      true,
		ProgramCheck: ruleMapUpdateNoKey,
	}

	// 8. Pointer bitwise/shift ops on pointer-like registers
	Rules["pointer_bitwise_shift"] = &Rule{
		Name:        "pointer_bitwise_shift",
		Description: "Detect bitwise and shift operations on registers that may hold pointers",
		Enabled:     true,
		BlockCheck:  rulePointerBitwiseShift,
	}

	// 9. Unknown helper index
	Rules["unknown_helper"] = &Rule{
		Name:         "unknown_helper",
		Description:  "Detect helper calls with unknown or unexpected helper indices",
		Enabled:      true,
		ProgramCheck: ruleUnknownHelper,
	}

	// 10. Missing BTF
	Rules["missing_btf"] = &Rule{
		Name:         "missing_btf",
		Description:  "Detect absence of BTF data in ELF which may cause verifier to reject loads/funcs",
		Enabled:      false, // disabled: needs ELF context not available in program-level checks
		ProgramCheck: ruleMissingBTF,
	}

	// 7. Complexity / path explosion
	Rules["high_complexity"] = &Rule{
		Name:         "high_complexity",
		Description:  "Block has too many instructions (path explosion risk)",
		Enabled:      true,
		ProgramCheck: ruleHighComplexity,
	}

	// STRICT ONLY — placeholders
	Rules["prog_insn_limit"] = &Rule{
		Name:         "prog_insn_limit",
		Description:  "Program instruction cap",
		Enabled:      false,
		ProgramCheck: func(blocks []cfg.BasicBlock) []string { return nil },
	}

	Rules["prog_block_limit"] = &Rule{
		Name:         "prog_block_limit",
		Description:  "Program block count limit",
		Enabled:      false,
		ProgramCheck: func(blocks []cfg.BasicBlock) []string { return nil },
	}

	Rules["prog_helper_limit"] = &Rule{
		Name:         "prog_helper_limit",
		Description:  "Total helper call limit",
		Enabled:      false,
		ProgramCheck: func(blocks []cfg.BasicBlock) []string { return nil },
	}
}

// =========================================================
// RULE IMPLEMENTATIONS
// =========================================================

func fmtmsg(s string, args ...interface{}) string {
	return fmt.Sprintf(s, args...)
}

// 1. Pointer arithmetic
// ------------------------------------------------------------
// Detect ALU arithmetic on R10 (frame pointer)
//
// NOTE: Without register state tracking, we can only reliably detect
// arithmetic on R10, which is ALWAYS a pointer and NEVER legally modified.
// R1 (ctx pointer) may be overwritten with scalars, causing false positives.
// ------------------------------------------------------------

func rulePointerArithmetic(block cfg.BasicBlock, ins parser.Instruction) []string {

	// Must be ALU or ALU64 class
	if !ins.IsALU() {
		return nil
	}

	// MOV operations replace the register, not arithmetic - skip them
	if ins.IsMove() {
		return nil
	}

	dst := ins.Dst()
	src := ins.Src()

	// R10 is ALWAYS the frame pointer and MUST be read-only
	// Any arithmetic on R10 is definitely illegal
	if dst == 10 {
		return []string{
			"CRITICAL: Pointer arithmetic on R10 (frame pointer)",
		}
	}

	// If R10 is used as source in arithmetic, also illegal
	// e.g., add r1, r10 → trying to create derived pointer
	if src == 10 {
		return []string{
			"CRITICAL: Pointer arithmetic using R10 as source (frame pointer)",
		}
	}

	// R1 (ctx pointer) is more complex:
	// - Initially holds context pointer
	// - But can be overwritten: r1 = 0, r1 = r2, etc.
	// Without state tracking, we can't tell if R1 still contains ctx pointer
	// So we DON'T check R1 to avoid false positives

	return nil
}

// 2. Helper chain
func ruleHelperChain(blocks []cfg.BasicBlock) []string {
	out := []string{}
	for _, b := range blocks {
		count := 0
		for _, ins := range b.Instructions {
			if ins.IsHelperCall() {
				count++
			}
		}
		if count > 1 {
			out = append(out,
				fmtmsg("Block %d: Multiple helper calls in a single block (may violate helper call chain)", b.ID))
		}
	}
	return out
}

// 3. Stack variable offsets
// ------------------------------------------------------------
// BPF stack access encoding:
//   STX: *(size *)(dst + off) = src    → dst=base, src=value, off=offset
//   LDX: dst = *(size *)(src + off)    → src=base, dst=result, off=offset
//
// Valid stack access: R10 + constant_offset where -512 <= offset <= -1
// Invalid: Variable offsets created by arithmetic on R10
// ------------------------------------------------------------

func ruleStackVarOffset(block cfg.BasicBlock, ins parser.Instruction) []string {

	// Only load/store instructions can access memory
	if !ins.IsLoad() && !ins.IsStore() {
		return nil
	}

	// Determine the base register:
	// - For STORE (STX): dst is the base pointer
	// - For LOAD (LDX): src is the base pointer
	baseReg := ins.Dst()
	if ins.IsLoad() {
		baseReg = ins.Src()
	}

	// We only care about stack accesses (R10 is stack pointer)
	if baseReg != 10 {
		return nil
	}

	// Check offset bounds: stack is [R10-512, R10-1]
	// Offset 0 would be R10 itself, which is invalid
	// Positive offsets or offsets < -512 are out of bounds
	off := ins.Off()
	if off < -512 || off > -1 {
		return []string{
			"Stack access uses invalid constant offset (unverifiable)",
		}
	}

	// NOTE: Variable offsets in BPF are created by doing arithmetic on R10:
	//   add r10, r1   ← This modifies R10 itself
	//   stx [r10], r2 ← Now uses modified R10
	//
	// This is caught by ruleWriteR10, not here.
	// The Src() field in STX is the VALUE being stored, not an index.

	return nil
}

// 4. Write to R10
// ------------------------------------------------------------
// R10 is the frame pointer and must remain read-only
// Any modification creates variable stack offsets
// ------------------------------------------------------------

func ruleWriteR10(block cfg.BasicBlock, ins parser.Instruction) []string {
	// ALU operations that modify R10 (add, sub, etc.)
	if ins.IsALU() && !ins.IsMove() && ins.Dst() == 10 {
		return []string{"CRITICAL: Arithmetic on R10 detected (frame pointer must be read-only)"}
	}

	// MOV operations that overwrite R10
	if ins.IsMove() && ins.Dst() == 10 {
		return []string{"CRITICAL: Write to R10 detected (frame pointer is read-only)"}
	}

	return nil
}

// 5. Map lookup without null check
// ------------------------------------------------------------
// After bpf_map_lookup_elem (helper 1), R0 contains pointer or NULL
// Must check R0 before dereferencing
// ------------------------------------------------------------

func ruleMapLookupNoNull(blocks []cfg.BasicBlock) []string {
	out := []string{}
	for _, b := range blocks {
		for i, ins := range b.Instructions {
			// Detect map lookup: helper call with imm=1
			if !ins.IsHelperCall() || ins.Imm != 1 {
				continue
			}

			// R0 will contain the result
			resultReg := 0
			checked := false

			// Look ahead in same block for null check
			// In BPF conditional jumps: if rX <op> rY/imm goto +off
			// The register being tested is Dst()
			for j := i + 1; j < len(b.Instructions); j++ {
				n := b.Instructions[j]
				if n.IsJump() && n.Dst() == resultReg {
					checked = true
					break
				}
			}

			if !checked {
				out = append(out,
					fmtmsg("MEDIUM: Block %d: Map lookup result used without null check", b.ID))
			}
		}
	}
	return out
}

// 6. Map update without key validation
// ------------------------------------------------------------
// NOTE: Cannot be properly implemented without register state tracking
// Keeping as heuristic warning for all map updates
// ------------------------------------------------------------

func ruleMapUpdateNoKey(blocks []cfg.BasicBlock) []string {
	out := []string{}
	for _, b := range blocks {
		for _, ins := range b.Instructions {
			// Detect map update: helper call with imm=2
			if ins.IsHelperCall() && ins.Imm == 2 {
				// Heuristic warning - cannot verify R2 contents without state tracking
				out = append(out,
					fmtmsg("MEDIUM: Block %d: Map update detected (ensure key in R2 is validated)", b.ID))
			}
		}
	}
	return out
}

// 7. High block complexity
func ruleHighComplexity(blocks []cfg.BasicBlock) []string {
	const complexityThreshold = 50
	out := []string{}
	for _, b := range blocks {
		if len(b.Instructions) > complexityThreshold {
			out = append(out,
				fmtmsg("Block %d: High block complexity (%d instructions, may cause verifier path explosion)",
					b.ID, len(b.Instructions)))
		}
	}
	return out
}

// Pointer bitwise/shift detection
// Flags ALU ops that include bitwise or shift on pointer-like registers
func rulePointerBitwiseShift(block cfg.BasicBlock, ins parser.Instruction) []string {
	if !ins.IsALU() || ins.IsMove() {
		return nil
	}

	// Check if the operation is one that is unsafe on pointers (AND/OR/XOR/LSH/RSH)
	if !ins.IsPtrArithmetic() {
		return nil
	}

	dst := ins.Dst()
	src := ins.Src()

	// R10 is always frame pointer and must not be subject to bitwise/shift ops
	if dst == 10 {
		return []string{"CRITICAL: Bitwise/shift operation on R10 (frame pointer)"}
	}
	if src == 10 {
		return []string{"CRITICAL: Bitwise/shift operation using R10 (frame pointer)"}
	}

	// Conservative: do not flag R1 (ctx ptr) to avoid false positives without state tracking
	return nil
}

// Program-level aggregation for R10 writes — higher severity
func ruleWriteR10Program(blocks []cfg.BasicBlock) []string {
	out := []string{}
	for _, b := range blocks {
		for _, ins := range b.Instructions {
			// Reuse the block-level detector
			msgs := ruleWriteR10(b, ins)
			for _, m := range msgs {
				// Promote message to program-level critical entry
				out = append(out, fmtmsg("CRITICAL: Block %d: %s", b.ID, m))
			}
		}
	}
	return out
}

// Unknown helper detection
// Heuristic: flag helper indices not in a conservative whitelist.
func ruleUnknownHelper(blocks []cfg.BasicBlock) []string {
	out := []string{}

	// Conservative whitelist of common helpers (kernel-dependent).
	known := map[int]bool{
		1:  true, // bpf_map_lookup_elem
		2:  true, // bpf_map_update_elem
		3:  true, // bpf_map_delete_elem
		4:  true,
		5:  true,
		6:  true,
		7:  true,
		8:  true,
		9:  true,
		10: true,
		11: true,
		12: true,
	}

	for _, b := range blocks {
		for _, ins := range b.Instructions {
			if !ins.IsHelperCall() {
				continue
			}
			hid := int(ins.Imm)
			if !known[hid] {
				out = append(out, fmtmsg("Block %d: Helper call with unknown index %d (imm=%d)", b.ID, hid, hid))
			}
		}
	}

	return out
}

// Missing BTF (placeholder)
// We cannot reliably detect missing BTF here because `ProgramCheck` only receives blocks
// which do not include ELF metadata. Keep as a no-op placeholder for future enhancements.
func ruleMissingBTF(blocks []cfg.BasicBlock) []string {
	return nil
}
