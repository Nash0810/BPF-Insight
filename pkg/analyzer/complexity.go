package analyzer

import (
	"fmt"
	"math"
	"strings"
	"time"
	// Ensure these imports are correct for your project structure
	"github.com/Nash0810/BPF-Insight/pkg/cfg"
	"github.com/Nash0810/BPF-Insight/pkg/parser"
	"github.com/Nash0810/BPF-Insight/pkg/verify"
	"github.com/cilium/ebpf/asm"
)

// ComplexityReport is the final output structure containing all metrics, score, and recommendations.
// (Section 11.1)
type ComplexityReport struct {
	// File info
	FilePath  string    `json:"file"`
	Section   string    `json:"section"`
	Timestamp time.Time `json:"timestamp"`

	// Basic metrics (From Instruction List)
	InstructionCount int `json:"instruction_count"`
	HelperCallCount  int `json:"helper_call_count"`

	// CFG metrics (From CFG)
	BasicBlockCount int     `json:"basic_block_count"`
	LoopCount       int     `json:"loops_detected"`
	MaxDepth        int     `json:"max_depth"`
	AvgBranching    float64 `json:"avg_branching"`

	// Scoring Components (0-100)
	CFGScore    float64 `json:"cfg_score"`    // Score from Instruction/CFG metrics
	RulePenalty float64 `json:"rule_penalty"` // Penalty from rule severity
	TotalScore  float64 `json:"TotalScore"`

	// Prediction
	Prediction string `json:"Prediction"`

	// Details
	Hotspots        []BlockComplexity `json:"hotspots"`
	Recommendations []Recommendation  `json:"recommendations"`
	CFG             *cfg.CFG          `json:"-"` // Exclude from JSON output
}

// BlockComplexity details the score for an individual basic block (for Hotspots)
// (Section 11.1)
type BlockComplexity struct {
	Block       *cfg.BasicBlock `json:"-"`
	BlockID     int             `json:"block_id"`
	OffsetRange string          `json:"offset_range"` // "start-end"
	Score       float64         `json:"score"`
	Reason      string          `json:"reason"` // Primary reason for high score
}

// Recommendation maps directly from verify.Violation
// (Section 11.1)
type Recommendation struct {
	Pattern    string `json:"pattern"`
	Severity   string `json:"severity"`
	Location   string `json:"location"`
	Issue      string `json:"issue"`
	Suggestion string `json:"suggestion"`
	Priority   int    `json:"-"` // Not for JSON, just for sorting
}

// Analyze runs the full complexity analysis pipeline.
func Analyze(filePath string, insts []asm.Instruction, progCFG *cfg.CFG, profileName string) (*ComplexityReport, error) {
	report := &ComplexityReport{
		FilePath:         filePath,
		Timestamp:        time.Now(),
		CFG:              progCFG,
		InstructionCount: len(insts),
		BasicBlockCount:  len(progCFG.Blocks),
		LoopCount:        len(progCFG.BackEdges),
		HelperCallCount:  calculateHelperCalls(insts), // Assuming a function exists
	}

	// 1. Calculate CFG Metrics and Per-Block Scores (Hotspots)
	cfgMetrics, hotspots := cfg.CalculateScores(progCFG, insts)
	report.MaxDepth = cfgMetrics.MaxDepth
	report.AvgBranching = cfgMetrics.AvgBranching

	// Convert cfg.BlockComplexity to analyzer.BlockComplexity
	report.Hotspots = make([]BlockComplexity, len(hotspots))
	for i, h := range hotspots {
		report.Hotspots[i] = BlockComplexity{
			BlockID:     h.Block.ID,
			OffsetRange: h.OffsetRange,
			Score:       h.ComplexityScore,
			Reason:      h.Reason,
		}
	}

	// 2. Run Rule Engine (Verifier)
	loops := cfg.DetectLoops(progCFG)
	
	// Empty program (no instructions) → guaranteed FAIL
	if len(insts) == 0 {
		report.TotalScore = 100.0
		report.CFGScore = 40.0
		report.RulePenalty = 60.0
		report.Prediction = "WILL_FAIL"
		report.Recommendations = []Recommendation{
			{
				Pattern:    "unparseable",
				Severity:   "critical",
				Location:   "Program",
				Issue:      "Empty or unparseable eBPF section",
				Suggestion: "Check ELF file structure and section headers",
				Priority:   1,
			},
		}
		return report, nil
	}

	// Convert []*BasicBlock to []BasicBlock for VerifyProgram
	blocks := make([]cfg.BasicBlock, len(progCFG.Blocks))
	for i, blk := range progCFG.Blocks {
		blocks[i] = *blk
	}

	// Build program metadata and pass to verifier
	hasBTF, _ := parser.HasBTF(filePath)
	meta := &verify.ProgramMeta{HasBTF: hasBTF, FilePath: filePath}
	vReport := verify.VerifyProgram(blocks, loops, meta)

	// Convert verify.VerifyOutput to analyzer.Recommendation
	report.Recommendations = convertVerifyOutputToRecommendations(vReport, progCFG)
	report.RulePenalty = calculateRulePenaltyFromWarnings(vReport)

	// 3. Calculate Final Total Score (I3 Integration)
	report.TotalScore, report.CFGScore = calculateTotalScore(report)

	// 4. Determine Final Prediction
	report.Prediction = getPrediction(report.TotalScore)

	return report, nil
}

// Helper: Calculate total BPF_CALL instructions
func calculateHelperCalls(insts []asm.Instruction) int {
	count := 0
	for _, ins := range insts {
		// BPF_CALL is opcode 0x85
		// Class = JMP (0x05), upper bits = 0x80
		opcode := uint8(ins.OpCode)
		if opcode == 0x85 {
			count++
		}
	}
	return count
}

// calculateRulePenaltyFromWarnings aggregates penalty points based on warnings.
func calculateRulePenaltyFromWarnings(vReport verify.VerifyOutput) float64 {
	penalty := 0.0

	// Severity mapping (points)
	// CRITICAL: 15, HIGH: 10, MEDIUM: 5, LOW: 2
	for _, bw := range vReport.BlockWarnings {
		msg := strings.ToLower(bw.Message)
		if strings.Contains(msg, "critical") {
			penalty += 15.0
		} else if strings.Contains(msg, "pointer arithmetic") || strings.Contains(msg, "frame pointer") || strings.Contains(msg, "bitwise/shift") || strings.Contains(msg, "bitwise") || strings.Contains(msg, "shift amount") {
			penalty += 10.0
		} else if strings.Contains(msg, "stack access") || strings.Contains(msg, "null check") || strings.Contains(msg, "unknown helper") {
			penalty += 5.0
		} else {
			penalty += 2.0
		}
	}

	// Program warnings: each is impactful
	for _, pw := range vReport.ProgramWarnings {
		// Program-level CRITICAL markers increase weight
		if strings.Contains(strings.ToLower(pw), "critical") {
			penalty += 20.0
		} else {
			penalty += 12.0
		}
	}

	// Cap rule penalty at 75 points (increased from 50)
	return math.Min(penalty, 75.0)
}

// convertVerifyOutputToRecommendations converts verify.VerifyOutput to analyzer.Recommendations
func convertVerifyOutputToRecommendations(vReport verify.VerifyOutput, progCFG *cfg.CFG) []Recommendation {
	var recommendations []Recommendation

	// Convert block warnings
	for _, bw := range vReport.BlockWarnings {
		sev := detectSeverityFromMessage(bw.Message)
		recommendations = append(recommendations, Recommendation{
			Pattern:    "block_warning",
			Severity:   sev,
			Location:   fmt.Sprintf("Block %d", bw.BlockID),
			Issue:      bw.Message,
			Suggestion: getSuggestion(bw.Message),
			Priority:   priorityFromSeverity(sev),
		})
	}

	// Convert program warnings
	for _, pw := range vReport.ProgramWarnings {
		sev := detectSeverityFromMessage(pw)
		recommendations = append(recommendations, Recommendation{
			Pattern:    "program_warning",
			Severity:   sev,
			Location:   "Program-level",
			Issue:      pw,
			Suggestion: getSuggestion(pw),
			Priority:   priorityFromSeverity(sev),
		})
	}

	return recommendations
}

// detectSeverityFromMessage maps message content to a severity label
func detectSeverityFromMessage(msg string) string {
	m := strings.ToLower(msg)
	if strings.Contains(m, "critical") {
		return "critical"
	}
	if strings.Contains(m, "pointer") || strings.Contains(m, "frame pointer") || strings.Contains(m, "bitwise") {
		return "high"
	}
	if strings.Contains(m, "stack") || strings.Contains(m, "null check") || strings.Contains(m, "unknown helper") {
		return "medium"
	}
	return "low"
}

func priorityFromSeverity(sev string) int {
	switch sev {
	case "critical":
		return 1
	case "high":
		return 2
	case "medium":
		return 3
	default:
		return 4
	}
}

// getSuggestion returns actionable suggestion text for known patterns
func getSuggestion(msg string) string {
	m := strings.ToLower(msg)
	switch {
	case strings.Contains(m, "r10") || strings.Contains(m, "frame pointer"):
		return "Never modify R10. Use constant offsets from R10 for stack access or use helpers."
	case strings.Contains(m, "pointer arithmetic"):
		return "Avoid arithmetic on pointer registers. Use helper-provided offsets or validated constants."
	case strings.Contains(m, "map lookup"):
		return "Check R0 for NULL after bpf_map_lookup_elem before dereferencing."
	case strings.Contains(m, "map update"):
		return "Validate key in R2 before bpf_map_update_elem."
	case strings.Contains(m, "unknown helper"):
		return "Confirm helper index and required BTF/compatibility for the program type."
	default:
		return "Review the instruction sequence and compare with kernel verifier error messages."
	}
}

// calculateTotalScore implements the final composite score formula.
// TotalScore = CFGScore + RulePenalty (Adjusted formula from plan)
func calculateTotalScore(r *ComplexityReport) (totalScore, cfgScore float64) {
	// Instruction Count Score (40 points cap)
	instructionScore := math.Min(float64(r.InstructionCount)/1000000.0*40.0, 40.0)

	// CFG Metrics Score Components
	// Max Depth (15 points cap)
	maxDepthWeight := 100.0 // Heuristic: Assume 100 is max reasonable depth
	depthScore := math.Min(float64(r.MaxDepth)/maxDepthWeight*15.0, 15.0)

	// Loop Complexity Score (10 points cap)
	loopScore := math.Min(float64(r.LoopCount)*5.0, 10.0)

	// Avg Branching Score (10 points cap)
	avgBranchingScore := math.Min(r.AvgBranching*5.0, 10.0) // 5.0 points per avg successor

	// Helper Call Score (5 points cap)
	helperScore := math.Min(float64(r.HelperCallCount)/50.0*5.0, 5.0)

	// Total CFG Score (Max 40+15+10+10+5 = 80)
	cfgScore = instructionScore + depthScore + loopScore + avgBranchingScore + helperScore

	// Final Score: CFG Complexity + Rule Penalties (Max 80 + 60 = 140, capped at 100)
	totalScore = math.Min(100.0, cfgScore+r.RulePenalty)

	// Round scores for cleaner output
	cfgScore = math.Round(cfgScore*10) / 10
	totalScore = math.Round(totalScore*10) / 10

	return totalScore, cfgScore
}

// getPrediction maps the final score to a categorical prediction.
// (Section 3.2 Interpretation)
func getPrediction(score float64) string {
	switch {
	case score < 25:
		return "LIKELY_PASS"
	case score < 50:
		return "MAY_PASS"
	case score < 75:
		return "LIKELY_FAIL"
	default:
		return "WILL_FAIL"
	}
}

// BatchResult represents the analysis result for a single file in batch processing.
type BatchResult struct {
	File       string  `json:"file"`
	Score      float64 `json:"score"`
	Prediction string  `json:"prediction"`
	Error      string  `json:"error,omitempty"`
}

// BatchReport contains the results of batch processing multiple files.
type BatchReport struct {
	TotalFiles int           `json:"total_files"`
	Analyzed   int           `json:"analyzed"`
	Errors     int           `json:"errors"`
	Results    []BatchResult `json:"results"`
	HighRisk   []string      `json:"high_risk"`
	AvgScore   float64       `json:"avg_score"`
}
