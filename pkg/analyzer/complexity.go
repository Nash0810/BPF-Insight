package analyzer

import (
	"math"
	"time"
	// Ensure these imports are correct for your project structure
	"github.com/Nash0810/BPF-Insight/pkg/cfg" 
	"github.com/Nash0810/BPF-Insight/pkg/verify"
	"github.com/cilium/ebpf/asm"
)

// ComplexityReport is the final output structure containing all metrics, score, and recommendations.
// (Section 11.1)
type ComplexityReport struct {
	// File info
	FilePath    string        `json:"file"`
	Section     string        `json:"section"`
	Timestamp   time.Time     `json:"timestamp"`

	// Basic metrics (From Instruction List)
	InstructionCount int `json:"instruction_count"`
	HelperCallCount  int `json:"helper_call_count"`

	// CFG metrics (From CFG)
	BasicBlockCount  int     `json:"basic_block_count"`
	LoopCount        int     `json:"loops_detected"`
	MaxDepth         int     `json:"max_depth"`
	AvgBranching     float64 `json:"avg_branching"`

	// Scoring Components (0-100)
	CFGScore         float64 `json:"cfg_score"`         // Score from Instruction/CFG metrics
	RulePenalty      float64 `json:"rule_penalty"`      // Penalty from rule severity
	TotalScore       float64 `json:"score"`
	
	// Prediction
	Prediction       string  `json:"prediction"`
	
	// Details
	Hotspots         []BlockComplexity `json:"hotspots"`
	Recommendations  []Recommendation  `json:"recommendations"`
	CFG              *cfg.CFG          `json:"-"` // Exclude from JSON output
}

// BlockComplexity details the score for an individual basic block (for Hotspots)
// (Section 11.1)
type BlockComplexity struct {
	Block            *cfg.BasicBlock `json:"-"`
	BlockID          int             `json:"block_id"`
	OffsetRange      string          `json:"offset_range"` // "start-end"
	Score            float64         `json:"score"`
	Reason           string          `json:"reason"` // Primary reason for high score
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
		FilePath: filePath,
		Timestamp: time.Now(),
		CFG: progCFG,
		InstructionCount: len(insts),
		BasicBlockCount: len(progCFG.Blocks),
		LoopCount: len(progCFG.BackEdges),
		HelperCallCount: calculateHelperCalls(insts), // Assuming a function exists
	}

	// 1. Calculate CFG Metrics and Per-Block Scores (Hotspots)
	cfgMetrics, hotspots := cfg.CalculateScores(progCFG, insts)
	report.MaxDepth = cfgMetrics.MaxDepth
	report.AvgBranching = cfgMetrics.AvgBranching
	
	// Convert cfg.BlockComplexity to analyzer.BlockComplexity
	report.Hotspots = make([]BlockComplexity, len(hotspots))
	for i, h := range hotspots {
		report.Hotspots[i] = BlockComplexity{
			BlockID: h.Block.ID,
			OffsetRange: h.OffsetRange,
			Score: h.ComplexityScore,
			Reason: h.Reason,
		}
	}

	// 2. Run Rule Engine (Verifier) - Your implemented feature
	vReport, err := verify.RunVerification(progCFG, profileName)
	if err != nil {
		return nil, err
	}
	
	// Convert verify.Violation to analyzer.Recommendation
	report.Recommendations = make([]Recommendation, len(vReport.Violations))
	report.RulePenalty = calculateRulePenalty(vReport.Violations)

	for i, v := range vReport.Violations {
		report.Recommendations[i] = Recommendation{
			Pattern: v.RuleName,
			Severity: v.Severity.String(),
			Location: v.Location,
			Issue: v.Issue,
			Suggestion: v.Suggestion,
			Priority: v.Severity.Priority(),
		}
	}

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
		if ins.OpCode.IsLoadStore() || ins.OpCode.IsArithmetic() || ins.OpCode.IsJump() {
			continue
		}
		// BPF_CALL is 0x85
		if ins.OpCode.Code() == 0x85 { 
			count++
		}
	}
	return count
}

// calculateRulePenalty aggregates penalty points based on rule severity.
func calculateRulePenalty(violations []verify.Violation) float64 {
	penalty := 0.0
	for _, v := range violations {
		// Penalties based on severity for integration with 0-100 score
		switch v.Severity {
		case verify.High:
			penalty += 15.0
		case verify.Medium:
			penalty += 8.0
		case verify.Low:
			penalty += 2.0
		}
	}
	// Cap rule penalty at 40 points (Section 3.2, Factor 1)
	return math.Min(penalty, 40.0) 
}

// calculateTotalScore implements the final composite score formula.
// TotalScore = CFGScore + RulePenalty (Adjusted formula from plan)
func calculateTotalScore(r *ComplexityReport) (totalScore, cfgScore float64) {
	// Instruction Count Score (40 points cap)
	instructionScore := math.Min(float64(r.InstructionCount) / 1000000.0 * 40.0, 40.0)

	// CFG Metrics Score Components
	// Max Depth (15 points cap) 
	maxDepthWeight := 100.0 // Heuristic: Assume 100 is max reasonable depth
	depthScore := math.Min(float64(r.MaxDepth) / maxDepthWeight * 15.0, 15.0)

	// Loop Complexity Score (10 points cap)
	loopScore := math.Min(float64(r.LoopCount) * 5.0, 10.0)
	
	// Avg Branching Score (10 points cap)
	avgBranchingScore := math.Min(r.AvgBranching * 5.0, 10.0) // 5.0 points per avg successor
	
	// Helper Call Score (5 points cap)
	helperScore := math.Min(float64(r.HelperCallCount) / 50.0 * 5.0, 5.0)

	// Total CFG Score (Max 40+15+10+10+5 = 80)
	cfgScore = instructionScore + depthScore + loopScore + avgBranchingScore + helperScore

	// Final Score: CFG Complexity + Rule Penalties (Max 80 + 40 = 120, capped at 100)
	totalScore = math.Min(100.0, cfgScore + r.RulePenalty)
	
	// Round scores for cleaner output
	cfgScore = math.Round(cfgScore*10)/10
	totalScore = math.Round(totalScore*10)/10
	
	return totalScore, cfgScore
}

// getPrediction maps the final score to a categorical prediction.
// (Section 3.2 Interpretation)
func getPrediction(score float64) string {
	switch {
	case score < 40:
		return "LIKELY_PASS"
	case score < 70:
		return "MAY_PASS"
	case score < 90:
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
	TotalFiles int            `json:"total_files"`
	Analyzed   int            `json:"analyzed"`
	Errors     int            `json:"errors"`
	Results    []BatchResult  `json:"results"`
	HighRisk   []string       `json:"high_risk"`
	AvgScore   float64        `json:"avg_score"`
}