package cfg

import (
	"fmt"
	"strings"
	// Removed the import of "bpf-insight/pkg/analyzer" to break the cycle.
)

// HotspotMap is a map of BlockID -> Score (defined here to break the import cycle)
type HotspotMap map[int]float64

// Constants for DOT color scheme (Section 5.7 C3)
const (
	ColorLowComplexity    = "lightgreen"
	ColorMediumComplexity = "yellow"
	ColorHighComplexity   = "orange"
	ColorCriticalComplexity = "red"
)

// GenerateDOT creates a Graphviz DOT file string for the CFG, using complexity scores for coloring.
// It now accepts the scores directly via HotspotMap to break the import cycle.
// (Section 5.7 C3)
func (progCFG *CFG) GenerateDOT(scoreMap HotspotMap) string {
	var sb strings.Builder
	
	sb.WriteString("digraph cfg {\n")
	sb.WriteString("  rankdir=TB;\n")
	sb.WriteString("  node [shape=box, style=filled];\n\n")

	// Node definitions
	for _, block := range progCFG.Blocks {
		score := scoreMap[block.ID]
		if score == 0 && progCFG.Entry != nil && block.ID == progCFG.Entry.ID {
			score = 0.1 // Ensure entry block has a score, even if minimal
		}
		color := getBlockColor(score)
		
		// Offsets are in bytes, divide by 8 for instruction number
		offsetRange := fmt.Sprintf("Insns: %d-%d\\n", block.StartOffset/8, block.EndOffset/8)
		
		// Detailed label
		label := fmt.Sprintf("Block %d\\n%sScore: %.1f",
			block.ID,
			offsetRange,
			score,
		)

		sb.WriteString(fmt.Sprintf("  block_%d [label=\"%s\", fillcolor=\"%s\"];\n",
			block.ID, label, color))
	}
	
	sb.WriteString("\n")
	
	// Edge definitions
	for _, edge := range progCFG.Edges {
		style := ""
		color := "black"
		label := string(edge.Type)
	
		if edge.Type == EdgeBackEdge {
			style = ", style=\"dashed\", color=\"red\""
			label = "back-edge (loop)"
		}
		
		sb.WriteString(fmt.Sprintf("  block_%d -> block_%d [label=\"%s\"%s];\n",
			edge.From.ID, edge.To.ID, label, style))
	}
	
	sb.WriteString("}\n")
	return sb.String()
}

// getBlockColor maps the per-block score to a color (Section 5.7 C3)
func getBlockColor(score float64) string {
	switch {
	case score < 10:
		return ColorLowComplexity // lightgreen
	case score < 30:
		return ColorMediumComplexity // yellow
	case score < 50:
		return ColorHighComplexity // orange
	default:
		return ColorCriticalComplexity // red
	}
}