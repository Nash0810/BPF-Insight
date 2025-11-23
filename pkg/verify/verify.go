package verify

import (
    "encoding/json"
    "fmt"

    "github.com/yourusername/bpfinsight/pkg/parser"
    "github.com/yourusername/bpfinsight/pkg/cfg"
)

// -------------------------------
// Result Data Types
// -------------------------------

type BlockWarning struct {
    BlockID int    `json:"BlockID"`
    Message string `json:"Message"`
}

type ProgramWarning struct {
    Message string `json:"Message"`
}

type VerifyResult struct {
    BlockWarnings   []BlockWarning   `json:"BlockWarnings"`
    ProgramWarnings []ProgramWarning `json:"ProgramWarnings"`
    FinalPrediction string           `json:"FinalPrediction"`
}

// -------------------------------
// JSON Output Helper
// -------------------------------

func ToJSON(v VerifyResult) ([]byte, error) {
    return json.MarshalIndent(v, "", "  ")
}

// -------------------------------
// Main Verifier Entry
// -------------------------------

func RunVerifier(insns []parser.Instruction, graph *cfg.CFG) VerifyResult {
    blockWarnings := runBlockChecks(insns, graph)
    progWarnings := runProgramChecks(insns, graph)

    prediction := computePrediction(blockWarnings, progWarnings)

    return VerifyResult{
        BlockWarnings:   blockWarnings,
        ProgramWarnings: progWarnings,
        FinalPrediction: prediction,
    }
}

// -------------------------------
// Prediction Logic
// -------------------------------

func computePrediction(bw []BlockWarning, pw []ProgramWarning) string {
    score := len(bw)*3 + len(pw)*6

    switch {
    case score >= 30:
        return "HIGH RISK"
    case score >= 15:
        return "MEDIUM RISK"
    default:
        return "LOW RISK"
    }
}

// -------------------------------
// Block-Level Rule Engine
// -------------------------------

func runBlockChecks(insns []parser.Instruction, graph *cfg.CFG) []BlockWarning {
    var out []BlockWarning

    rc := NewRegChecker()

    for _, block := range graph.Blocks {

        helperCount := 0

        for _, ins := range block.Instructions {
            dst := ins.Dst()
            src := ins.Src()

            //
            // ----- POINTER RULES -----
            //

            // Rule: pointer arithmetic on pointer register
            if RuleRegistry["pointer_arithmetic"].Enabled &&
                ins.IsPtrArithmetic() &&
                rc.State.Regs[dst] == RegPtr {

                out = append(out, BlockWarning{
                    BlockID: block.ID,
                    Message: "Pointer arithmetic on pointer register",
                })
            }

            // Rule: write to R10
            if RuleRegistry["write_r10"].Enabled &&
                dst == 10 && !ins.IsMove() {

                out = append(out, BlockWarning{
                    BlockID: block.ID,
                    Message: "Write to R10 detected (frame pointer is read-only)",
                })
            }

            //
            // ----- STACK RULES -----
            //

            if RuleRegistry["stack_var_offset"].Enabled &&
                isVariableStackOffset(ins) {

                out = append(out, BlockWarning{
                    BlockID: block.ID,
                    Message: "Stack access uses variable offset (verifier cannot prove bounds)",
                })
            }

            //
            // ----- HELPER RULES -----
            //

            if ins.IsHelperCall() {
                helperCount++
            }

            //
            // ----- CONTROL FLOW RULES -----
            //

            if RuleRegistry["unknown_jump"].Enabled &&
                ins.IsJump() &&
                rc.State.Regs[src] == RegUnknown {

                out = append(out, BlockWarning{
                    BlockID: block.ID,
                    Message: "Conditional jump on unknown register state",
                })
            }

            //
            // Apply register state transitions last
            //
            rc.State.Apply(ins)
        }

        // Rule: multiple helper calls in one block
        if RuleRegistry["helper_chain"].Enabled &&
            helperCount >= 2 {

            out = append(out, BlockWarning{
                BlockID: block.ID,
                Message: "Multiple helper calls in a single block (may violate helper call chains)",
            })
        }

        // Rule: block complexity
        if RuleRegistry["high_complexity"].Enabled &&
            len(block.Instructions) >= 10 {

            out = append(out, BlockWarning{
                BlockID: block.ID,
                Message: "High block complexity (may trigger verifier path explosion)",
            })
        }
    }

    return out
}

// -------------------------------
// Program-Level Rules
// -------------------------------

func runProgramChecks(insns []parser.Instruction, graph *cfg.CFG) []ProgramWarning {
    var out []ProgramWarning

    // Future program-level rules go here.

    return out
}

// -------------------------------
// Utility Functions
// -------------------------------

// Detect variable stack offset (e.g., r10 - r2)
func isVariableStackOffset(ins parser.Instruction) bool {
    if !ins.IsStore() && !ins.IsLoad() {
        return false
    }

    // Offset must be register-based
    // eBPF encoding: variable offset means ins.OffsetVal == 0 and source is register
    if ins.OffsetVal == 0 && ins.Src() != 0 {
        return true
    }
    return false
}
