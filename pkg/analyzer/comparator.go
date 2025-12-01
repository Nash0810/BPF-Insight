package analyzer

import (
	"fmt"
	"math"
	"strings"
)

// ComparisonReport holds the results of comparing two ComplexityReports.
type ComparisonReport struct {
	Before           *ComplexityReport  `json:"before"`
	After            *ComplexityReport  `json:"after"`
	ScoreDelta       float64            `json:"score_delta"` // Positive = improvement (After score is lower)
	MetricChanges    map[string]float64 `json:"metric_changes"`
	ResolvedPatterns []Recommendation   `json:"resolved_patterns"`
	NewPatterns      []Recommendation   `json:"new_patterns"`
	Summary          string             `json:"summary"`
}

// ComparePrograms analyzes two reports and generates a diff.
func ComparePrograms(beforeReport, afterReport *ComplexityReport) *ComparisonReport {
	report := &ComparisonReport{
		Before: beforeReport,
		After:  afterReport,
		// Delta is Before - After. Positive value is an improvement (lower score after optimization)
		ScoreDelta:    math.Round((beforeReport.TotalScore-afterReport.TotalScore)*10) / 10,
		MetricChanges: make(map[string]float64),
	}

	// Calculate metric changes (Delta = Before - After)
	report.MetricChanges["instructions"] = float64(beforeReport.InstructionCount - afterReport.InstructionCount)
	report.MetricChanges["loops"] = float64(beforeReport.LoopCount - afterReport.LoopCount)
	report.MetricChanges["max_depth"] = float64(beforeReport.MaxDepth - afterReport.MaxDepth)
	report.MetricChanges["avg_branching"] = beforeReport.AvgBranching - afterReport.AvgBranching
	report.MetricChanges["rule_penalty"] = beforeReport.RulePenalty - afterReport.RulePenalty

	// Analyze Pattern Changes
	beforeMap := make(map[string]Recommendation)
	for _, r := range beforeReport.Recommendations {
		// Use Pattern + Location as unique key for a specific violation
		beforeMap[r.Pattern+r.Location] = r
	}

	afterMap := make(map[string]Recommendation)
	for _, r := range afterReport.Recommendations {
		afterMap[r.Pattern+r.Location] = r
	}

	// Identify Resolved and New Patterns
	for key, r := range beforeMap {
		if _, exists := afterMap[key]; !exists {
			report.ResolvedPatterns = append(report.ResolvedPatterns, r)
		}
	}
	for key, r := range afterMap {
		if _, exists := beforeMap[key]; !exists {
			report.NewPatterns = append(report.NewPatterns, r)
		}
	}

	// Generate Summary (Section 5.12, Example Output)
	if report.ScoreDelta > 10 {
		report.Summary = "Significant improvement: Complexity score reduced substantially."
	} else if report.ScoreDelta > 0.1 {
		report.Summary = "Minor improvement: Good reduction in complexity."
	} else if report.ScoreDelta < -10 {
		report.Summary = "Significant regression: Complexity score increased substantially."
	} else if report.ScoreDelta < -0.1 {
		report.Summary = "Minor regression detected."
	} else {
		report.Summary = "No significant change in complexity."
	}

	return report
}

// PrintComparisonText generates a human-readable comparison report.
// (Section 12.4)
func PrintComparisonText(report *ComparisonReport) string {
	var sb strings.Builder

	sb.WriteString("Comparative Analysis\n")
	sb.WriteString("====================\n")
	sb.WriteString(fmt.Sprintf("Before: %s\n", report.Before.FilePath))
	sb.WriteString(fmt.Sprintf("After:  %s\n", report.After.FilePath))
	sb.WriteString("\n")

	sb.WriteString("Complexity Scores:\n")
	sb.WriteString(fmt.Sprintf("  Before: %.1f / 100 (%s)\n", report.Before.TotalScore, report.Before.Prediction))
	sb.WriteString(fmt.Sprintf("  After:  %.1f / 100 (%s)\n", report.After.TotalScore, report.After.Prediction))

	deltaStr := "no change"
	if report.ScoreDelta > 0 {
		deltaStr = fmt.Sprintf("-%s%.1f (improvement)", "%", report.ScoreDelta)
	} else if report.ScoreDelta < 0 {
		deltaStr = fmt.Sprintf("+%.1f (regression)", math.Abs(report.ScoreDelta))
	}
	sb.WriteString(fmt.Sprintf("  Delta:  %s\n", deltaStr))
	sb.WriteString("\n")

	sb.WriteString("Metric Changes (Before -> After):\n")
	for key, delta := range report.MetricChanges {
		if math.Abs(delta) > 0.1 {
			beforeVal := getMetricValue(report.Before, key)
			afterVal := getMetricValue(report.After, key)
			percent := 0.0
			if beforeValFloat, ok := beforeVal.(float64); ok && beforeValFloat != 0 {
				percent = delta / beforeValFloat * 100
			}

			sb.WriteString(fmt.Sprintf("  %-20s %v -> %v (%.1f, %+.1f%%)\n",
				formatMetricKey(key), beforeVal, afterVal, delta, percent))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("Pattern Analysis:\n")
	if len(report.ResolvedPatterns) > 0 {
		sb.WriteString("  ✓ Resolved Issues:\n")
		for _, r := range report.ResolvedPatterns {
			sb.WriteString(fmt.Sprintf("    - [%s] %s at %s\n", r.Severity, r.Issue, r.Location))
		}
	}
	if len(report.NewPatterns) > 0 {
		sb.WriteString("  ⚠ New Issues:\n")
		for _, r := range report.NewPatterns {
			sb.WriteString(fmt.Sprintf("    - [%s] %s at %s\n", r.Severity, r.Issue, r.Location))
		}
	}
	if len(report.ResolvedPatterns) == 0 && len(report.NewPatterns) == 0 {
		sb.WriteString("  (No change in critical rule violations)\n")
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("Summary: %s\n", report.Summary))

	return sb.String()
}

// Utility functions for text output formatting
func formatMetricKey(key string) string {
	switch key {
	case "instructions":
		return "Instructions"
	case "loops":
		return "Loops"
	case "max_depth":
		return "Max Depth"
	case "avg_branching":
		return "Avg Branching"
	case "rule_penalty":
		return "Rule Penalty"
	default:
		return key
	}
}

func getMetricValue(r *ComplexityReport, key string) interface{} {
	switch key {
	case "instructions":
		return r.InstructionCount
	case "loops":
		return r.LoopCount
	case "max_depth":
		return r.MaxDepth
	case "avg_branching":
		return math.Round(r.AvgBranching*10) / 10
	case "rule_penalty":
		return math.Round(r.RulePenalty*10) / 10
	default:
		return 0.0
	}
}
