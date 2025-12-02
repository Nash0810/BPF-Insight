package verify

import (
	"encoding/json"

	"github.com/Nash0810/BPF-Insight/pkg/cfg"
)

// ProgramMeta carries additional program-level metadata useful for rules
type ProgramMeta struct {
	HasBTF   bool
	Section  string
	FilePath string
}

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

func VerifyProgram(blocks []cfg.BasicBlock, loops []cfg.LoopInfo, meta *ProgramMeta) VerifyOutput {
	var bw []BlockWarning
	var pw []string

	if len(blocks) == 0 {
		// No blocks, run program-level checks only
		for _, rule := range Rules {
			if !rule.Enabled || rule.ProgramCheck == nil {
				continue
			}
			msgs := rule.ProgramCheck(blocks)
			pw = append(pw, msgs...)
		}
		return VerifyOutput{BlockWarnings: bw, ProgramWarnings: pw, FinalPrediction: riskLevel(bw, pw)}
	}

	// Build block map and successor list (use IDs)
	blockMap := make(map[int]cfg.BasicBlock)
	succs := make(map[int][]int)
	var entryID int = blocks[0].ID
	for _, b := range blocks {
		blockMap[b.ID] = b
		if len(b.Predecessors) == 0 {
			entryID = b.ID
		}
		for _, s := range b.Successors {
			if s != nil {
				succs[b.ID] = append(succs[b.ID], s.ID)
			}
		}
	}

	// Worklist for dataflow (register-state at block entry)
	entryState := make(map[int]*RegState)
	for id := range blockMap {
		entryState[id] = nil
	}
	// Entry block initial state
	entryState[entryID] = NewRegState()
	worklist := []int{entryID}

	// Dataflow: forward propagation of RegState
	for len(worklist) > 0 {
		bID := worklist[0]
		worklist = worklist[1:]
		b, ok := blockMap[bID]
		if !ok {
			continue
		}

		inState := entryState[bID]
		if inState == nil {
			// unreachable or not yet initialized
			inState = NewUnknownState()
			entryState[bID] = inState
		}

		// Simulate instructions in block, applying rules with state
		cur := inState.Clone()
		for _, ins := range b.Instructions {
			for _, rule := range Rules {
				if !rule.Enabled {
					continue
				}
				// Prefer stateful check if available
				if rule.BlockCheckState != nil {
					msgs := rule.BlockCheckState(b, ins, cur)
					for _, m := range msgs {
						bw = append(bw, BlockWarning{BlockID: bID, Message: m})
					}
				} else if rule.BlockCheck != nil {
					msgs := rule.BlockCheck(b, ins)
					for _, m := range msgs {
						bw = append(bw, BlockWarning{BlockID: bID, Message: m})
					}
				}
			}

			// Apply instruction effects to current RegState
			cur.Apply(ins)
		}

		// Propagate to successors
		for _, sid := range succs[bID] {
			if entryState[sid] == nil {
				entryState[sid] = cur.Clone()
				worklist = append(worklist, sid)
			} else {
				// Merge and enqueue if changed
				changed := entryState[sid].Merge(cur)
				if changed {
					worklist = append(worklist, sid)
				}
			}
		}
	}

	// PROGRAM-LEVEL: run registered program checks
	for _, rule := range Rules {
		if !rule.Enabled {
			continue
		}
		if rule.ProgramCheck != nil {
			msgs := rule.ProgramCheck(blocks)
			pw = append(pw, msgs...)
		}
	}

	// Meta-aware program checks (missing BTF, unknown helpers using section)
	if meta != nil {
		// missing BTF
		msgs := ruleMissingBTFWithMeta(blocks, meta)
		pw = append(pw, msgs...)

		// unknown helper with meta.section
		msgs2 := ruleUnknownHelperWithMeta(blocks, meta)
		pw = append(pw, msgs2...)
	} else {
		// No meta: run legacy unknown helper check
		msgs := ruleUnknownHelper(blocks)
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
