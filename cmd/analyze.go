package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Nash0810/BPF-Insight/pkg/analyzer"
	"github.com/Nash0810/BPF-Insight/pkg/parser"
)

var outputJSON bool

func init() {
	analyzeCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "Output in JSON format")
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze <file>",
	Short: "Analyze eBPF program complexity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]

		// Parse ELF
		p := parser.ELFParser{FilePath: file}
		raw, err := p.Parse()
		if err != nil {
			return fmt.Errorf("ELF parse error: %w", err)
		}

		// Decode instructions
		insns, err := parser.DecodeInstructions(raw)
		if err != nil {
			return fmt.Errorf("decode error: %w", err)
		}

		// Score
		report := analyzer.ScoreInstructions(insns)

        // Output
		if outputJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(report)
		}

		fmt.Println("eBPF Verifier Complexity Analysis")
		fmt.Println("==================================")
		fmt.Printf("File: %s\n\n", file)
		fmt.Printf("Instructions:    %d\n", report.InstructionCount)
		fmt.Printf("Jumps:           %d\n", report.JumpCount)
		fmt.Printf("Helper Calls:    %d\n\n", report.HelperCallCount)
		fmt.Printf("Complexity Score: %.2f / 100\n", report.TotalScore)
		fmt.Printf("Prediction: %s\n", report.Prediction)

		return nil
	},
}
