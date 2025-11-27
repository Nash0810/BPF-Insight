package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/Nash0810/BPF-Insight/pkg/analyzer"
	"github.com/Nash0810/BPF-Insight/pkg/cfg"
	"github.com/Nash0810/BPF-Insight/pkg/parser"
	"github.com/Nash0810/BPF-Insight/pkg/utils"
)

var (
	outputJSON   bool
	verbose      bool
	showCFG      bool
	outputDir    string
	hotspotsNum  int
	noViz        bool
)

func init() {
	analyzeCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "Output in JSON format")
	analyzeCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed metrics")
	analyzeCmd.Flags().BoolVar(&showCFG, "show-cfg", false, "Generate CFG visualization")
	analyzeCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Directory for output files")
	analyzeCmd.Flags().IntVarP(&hotspotsNum, "hotspots", "n", 5, "Number of hotspots to show")
	analyzeCmd.Flags().BoolVar(&noViz, "no-viz", false, "Skip visualization generation")
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze <file>",
	Short: "Analyze eBPF program complexity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]

		// Use the shared analyzeSingleFile function
		report, err := analyzeSingleFile(file)
		if err != nil {
			return err
		}

		// Output
		if outputJSON {
			return utils.PrintJSON(report)
		}

		// Text output
		fmt.Println("eBPF Verifier Complexity Analysis")
		fmt.Println("==================================")
		fmt.Printf("File: %s\n", report.FilePath)
		if report.Section != "" {
			fmt.Printf("Section: %s\n", report.Section)
		}
		fmt.Println()

		fmt.Println("Metrics:")
		fmt.Printf("  Instructions:     %d\n", report.InstructionCount)
		fmt.Printf("  Basic Blocks:     %d\n", report.BasicBlockCount)
		if verbose {
			fmt.Printf("  Max Depth:        %d\n", report.MaxDepth)
			fmt.Printf("  Loops Detected:   %d\n", report.LoopCount)
			fmt.Printf("  Avg Branching:    %.1f\n", report.AvgBranching)
		}
		fmt.Printf("  Helper Calls:     %d\n", report.HelperCallCount)
		fmt.Println()

		fmt.Printf("Complexity Score: %.1f / 100\n", report.TotalScore)
		fmt.Printf("Prediction: %s\n", report.Prediction)
		fmt.Println()

		// Show hotspots if available
		if len(report.Hotspots) > 0 && (verbose || hotspotsNum > 0) {
			n := hotspotsNum
			if n > len(report.Hotspots) {
				n = len(report.Hotspots)
			}
			fmt.Printf("Top Complexity Hotspots:\n")
			for i := 0; i < n; i++ {
				hs := report.Hotspots[i]
				fmt.Printf("  %d. Block %d (insns %s) - Score: %.1f\n",
					i+1, hs.BlockID, hs.OffsetRange, hs.Score)
				if hs.Reason != "" {
					fmt.Printf("     └─ %s\n", hs.Reason)
				}
			}
			fmt.Println()
		}

		// Show recommendations
		if len(report.Recommendations) > 0 {
			// Sort by priority
			recs := make([]analyzer.Recommendation, len(report.Recommendations))
			copy(recs, report.Recommendations)
			sort.Slice(recs, func(i, j int) bool {
				return recs[i].Priority > recs[j].Priority
			})

			fmt.Println("Recommendations:")
			for _, rec := range recs {
				emoji := "🟡"
				switch rec.Severity {
				case "high", "critical":
					emoji = "🔴"
				case "medium":
					emoji = "🟠"
				}
				fmt.Printf("  %s %s: %s at %s\n", emoji, rec.Severity, rec.Issue, rec.Location)
				if rec.Suggestion != "" {
					fmt.Printf("     └─ %s\n", rec.Suggestion)
				}
			}
			fmt.Println()
		}

		// Generate CFG visualization if requested
		if showCFG && !noViz && report.CFG != nil {
			dir := outputDir
			if dir == "" {
				dir = "."
			}
			os.MkdirAll(dir, 0755)

			base := filepath.Base(file)
			dotFile := filepath.Join(dir, base+".dot")

			// Create hotspot map for visualization
			scoreMap := make(cfg.HotspotMap)
			for _, hs := range report.Hotspots {
				scoreMap[hs.BlockID] = hs.Score
			}

			dotContent := report.CFG.GenerateDOT(scoreMap)
			if err := os.WriteFile(dotFile, []byte(dotContent), 0644); err != nil {
				return fmt.Errorf("failed to write CFG file: %w", err)
			}

			fmt.Printf("CFG saved to: %s\n", dotFile)
			fmt.Println("Run 'dot -Tpng " + dotFile + " -o cfg.png' to visualize")
		}

		return nil
	},
}
