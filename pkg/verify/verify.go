package verify

import (
	"encoding/json"

	"github.com/Nash0810/BPF-Insight/pkg/cfg"
)

type BlockWarning struct {
	BlockID int    `json:"BlockID"`
	Message string `json:"Message"`
}

type VerifyOutput struct {
	BlockWarnings   []BlockWarning `json:"BlockWarnings"`
	ProgramWarnings []string       `json:"ProgramWarnings"`
	FinalPrediction string         `json:"FinalPrediction"`
}

func (v VerifyOutput) ToJSON() ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func VerifyProgram(blocks []cfg.BasicBlock, loops []cfg.LoopInfo) VerifyOutput {

	var bw []BlockWarning
	var pw []string

	// BLOCK-LEVEL
	for _, block := range blocks {
		for _, ins := range block.Instructions {
			for _, rule := range Rules {
				if !rule.Enabled || rule.BlockCheck == nil {
					continue
				}
				msgs := rule.BlockCheck(block, ins)
				for _, m := range msgs {
					bw = append(bw, BlockWarning{BlockID: block.ID, Message: m})
				}
			}
		}
	}

	// PROGRAM-LEVEL
	for _, rule := range Rules {
		if !rule.Enabled || rule.ProgramCheck == nil {
			continue
		}
		msgs := rule.ProgramCheck(blocks)
		pw = append(pw, msgs...)
	}

	return VerifyOutput{
		BlockWarnings:   bw,
		ProgramWarnings: pw,
		FinalPrediction: riskLevel(bw, pw),
	}
}

// Simple risk model
func riskLevel(bw []BlockWarning, pw []string) string {
	score := len(bw)*2 + len(pw)*4

	switch {
	case score == 0:
		return "LOW RISK"
	case score < 12:
		return "MEDIUM RISK"
	default:
		return "HIGH RISK"
	}
}
