package analyzer

import (
	"github.com/Nash0810/BPF-Insight/pkg/parser"
)

type ComplexityReport struct {
	InstructionCount int
	JumpCount        int
	HelperCallCount  int
	TotalScore       float64
	Prediction       string
}

func ScoreInstructions(insns []parser.Instruction) *ComplexityReport {
	report := &ComplexityReport{
		InstructionCount: len(insns),
	}

	// Count jumps + helper calls
	for _, ins := range insns {
		if ins.IsJump() {
			report.JumpCount++
		}
		if ins.Opcode == 0x85 { // Helper call
			report.HelperCallCount++
		}
	}

	// Basic scoring model (Iteration 1)
	instrScore := float64(report.InstructionCount) / 1_000_000 * 40
	branchScore := float64(report.JumpCount) / 100 * 30
	helperScore := float64(report.HelperCallCount) / 50 * 5

	total := instrScore + branchScore + helperScore
	if total > 100 {
		total = 100
	}
	report.TotalScore = total

	// Prediction buckets
	switch {
	case total < 40:
		report.Prediction = "LIKELY PASS"
	case total < 70:
		report.Prediction = "MAY PASS"
	case total < 90:
		report.Prediction = "LIKELY FAIL"
	default:
		report.Prediction = "WILL FAIL"
	}

	return report
}
