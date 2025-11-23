package verify

import (
	"github.com/yourusername/bpfinsight/pkg/cfg"
	"github.com/yourusername/bpfinsight/pkg/parser"
)

type BlockWarning struct {
	BlockID int
	Message string
}

type ProgramWarning struct {
	Message string
}

type VerifyResult struct {
	BlockWarnings   []BlockWarning
	ProgramWarnings []ProgramWarning
	FinalPrediction string
}

// ---------- Rule Categories (Option B: 25-rule engine) ----------

// Detect control-flow related issues
func checkControlFlow(graph *cfg.CFG, score cfg.ProgramScore) []BlockWarning {
	var warnings []BlockWarning

	for _, block := range graph.Blocks {

		// Rule: high-complexity block
		if score.Blocks[block.ID].Score > 40 {
			warnings = append(warnings, BlockWarning{
				BlockID: block.ID,
				Message: "High block complexity (may trigger verifier path explosion)",
			})
		}

		// Rule: block has too many successors
		if len(block.Successors) > 2 {
			warnings = append(warnings, BlockWarning{
				BlockID: block.ID,
				Message: "Branch fan-out > 2 (verifier struggles with wide branching)",
			})
		}
	}

	return warnings
}

// Detect helper-related patterns
func checkHelpers(insns []parser.Instruction, graph *cfg.CFG) []BlockWarning {
	var warnings []BlockWarning

	for _, block := range graph.Blocks {
		helperCount := 0
		for _, ins := range block.Instructions {
			if ins.IsHelperCall() {
				helperCount++
			}
		}

		// Rule: too many helpers in a single block
		if helperCount >= 3 {
			warnings = append(warnings, BlockWarning{
				BlockID: block.ID,
				Message: "Multiple helper calls in a single block (may violate helper call chains)",
			})
		}
	}

	return warnings
}

// Detect stack & memory issues
func checkMemory(insns []parser.Instruction, graph *cfg.CFG) []BlockWarning {
	var warnings []BlockWarning
	// Placeholder — rules added in Step 2
	return warnings
}

// Detect register safety problems
func checkRegisters(insns []parser.Instruction, graph *cfg.CFG) []BlockWarning {
	var warnings []BlockWarning
	// Placeholder — rules added in Step 3
	return warnings
}

// Detect map-use verification risks
func checkMaps(insns []parser.Instruction, graph *cfg.CFG) []BlockWarning {
	var warnings []BlockWarning
	// Placeholder — rules added in Step 4
	return warnings
}

// Detect global program-level risks
func checkGlobal(score cfg.ProgramScore, graph *cfg.CFG) []ProgramWarning {
	var pw []ProgramWarning

	// Global: program too complex
	if score.TotalScore > 70 {
		pw = append(pw, ProgramWarning{
			Message: "Overall program complexity is high (likely verifier rejection)",
		})
	}

	return pw
}

// ---------- Master Verification Engine ----------

func VerifyProgram(insns []parser.Instruction, graph *cfg.CFG, score cfg.ProgramScore) VerifyResult {

	blocks := []BlockWarning{}
	blocks = append(blocks, checkControlFlow(graph, score)...)
	blocks = append(blocks, checkHelpers(insns, graph)...)
	blocks = append(blocks, checkMemory(insns, graph)...)
	blocks = append(blocks, checkRegisters(insns, graph)...)
	blocks = append(blocks, checkMaps(insns, graph)...)

	prog := checkGlobal(score, graph)

	// Final prediction
	final := "LOW RISK"
	if score.TotalScore > 70 || len(prog) > 0 {
		final = "HIGH RISK"
	} else if score.TotalScore > 50 || len(blocks) > 3 {
		final = "MEDIUM RISK"
	}

	return VerifyResult{
		BlockWarnings:   blocks,
		ProgramWarnings: prog,
		FinalPrediction: final,
	}
}
