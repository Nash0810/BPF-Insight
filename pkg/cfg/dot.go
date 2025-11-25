package cfg

import (
    "fmt"
    "os"
    "strings"

    "github.com/Nash0810/BPF-Insight/pkg/parser"
)


func summarizeInstructions(insns []parser.Instruction, max int) string {
	var lines []string
	limit := max
	if len(insns) < max {
		limit = len(insns)
	}

	for i := 0; i < limit; i++ {
        ins := insns[i]
        var line string

        if ins.IsHelperCall() {
            line = fmt.Sprintf("call helper %d", ins.Imm)
        } else if ins.IsJump() {
            line = fmt.Sprintf("if r%d jmp r%d %+d", ins.Src(), ins.Dst(), ins.OffsetVal)
        } else {
            line = fmt.Sprintf("op 0x%x r%d,r%d imm=%d", ins.Opcode, ins.Dst(), ins.Src(), ins.Imm)
        }

        lines = append(lines, line)
	}

	if len(insns) > max {
		lines = append(lines, "...")
	}

	return strings.Join(lines, "\\l") + "\\l"
}

// Map score to risk color
func colorForScore(score float64) string {
	switch {
	case score < 10:
		return "#c8e6c9" // Light green
	case score < 25:
		return "#fff9c4" // Yellow
	case score < 50:
		return "#ffe082" // Orange
	default:
		return "#ffab91" // Red
	}
}

// Enhanced DOT with scoring, color, header, legend
func (cfg *CFG) WriteDOTScored(filename string, loops []LoopInfo, score ProgramScore) error {

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "digraph CFG {\n")
	fmt.Fprintf(f, "  node [shape=box, style=filled, fontname=\"Courier\"];\n")
	fmt.Fprintf(f, "  labelloc=\"t\";\n")
	fmt.Fprintf(f, "  label=\"Program Score: %.2f  |  Prediction: %s\";\n",
		score.TotalScore, score.Prediction)

	// Legend
	fmt.Fprintf(f, "  subgraph cluster_legend {\n")
	fmt.Fprintf(f, "    label=\"Risk Legend\";\n")
	fmt.Fprintf(f, "    fontsize=12;\n")
	fmt.Fprintf(f,
		"    L1 [label=\"Low (0–10)\", fillcolor=\"#c8e6c9\"];\n"+
			"    L2 [label=\"Medium (10–25)\", fillcolor=\"#fff9c4\"];\n"+
			"    L3 [label=\"High (25–50)\", fillcolor=\"#ffe082\"];\n"+
			"    L4 [label=\"Critical (50+)\", fillcolor=\"#ffab91\"];\n")
	fmt.Fprintf(f, "  }\n")

	// Build nodes with scoring
	scoreMap := map[int]BlockScore{}
	for _, b := range score.Blocks {
		scoreMap[b.BlockID] = b
	}

	for _, block := range cfg.Blocks {

		bs := scoreMap[block.ID]
		color := colorForScore(bs.Score)

		// Summaries
		summary := summarizeInstructions(block.Instructions, 3)

		label := fmt.Sprintf(
			"Block B%d\\nScore: %.2f\\ninsns: %d | jumps: %d | helpers: %d\\n\\n%s",
			block.ID, bs.Score, bs.Insns, bs.Jumps, bs.Helpers, summary,
		)

		fmt.Fprintf(f, "  B%d [label=\"%s\", fillcolor=\"%s\"];\n",
			block.ID, label, color)
	}

	// Edges
	for _, block := range cfg.Blocks {
		for _, succ := range block.Successors {
			fmt.Fprintf(f, "  B%d -> B%d;\n", block.ID, succ)
		}
	}

	fmt.Fprintf(f, "}\n")
	return nil
}
