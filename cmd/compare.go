package main

import (
	"fmt"
	"os"

	"github.com/Nash0810/BPF-Insight/pkg/analyzer"
	"github.com/Nash0810/BPF-Insight/pkg/cfg"
	"github.com/Nash0810/BPF-Insight/pkg/parser"
	"github.com/Nash0810/BPF-Insight/pkg/utils"
	"github.com/spf13/cobra"
)

// Global variable for profile name (can be set via flag in future)
var profileName string

var compareCmd = &cobra.Command{
	Use:   "compare <before.o> <after.o>",
	Short: "Compare two eBPF program versions for complexity changes",
	Long: `Compares the complexity report of a 'before' version and an 'after' (optimized) version.
It highlights score changes, metric deltas, and resolved/new rule violations.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		beforeFile := args[0]
		afterFile := args[1]
		outputFormat, _ := cmd.Flags().GetString("output-format")

		beforeReport, err := analyzeSingleFile(beforeFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing BEFORE file (%s): %v\n", beforeFile, err)
			os.Exit(1)
		}

		afterReport, err := analyzeSingleFile(afterFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing AFTER file (%s): %v\n", afterFile, err)
			os.Exit(1)
		}

		compareReport := analyzer.ComparePrograms(beforeReport, afterReport)

		switch outputFormat {
		case "json":
			utils.PrintJSON(compareReport)
		case "text":
			fmt.Println(analyzer.PrintComparisonText(compareReport))
		default:
			fmt.Fprintf(os.Stderr, "Unsupported output format: %s\n", outputFormat)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(compareCmd)
	compareCmd.Flags().StringP("output-format", "f", "text", "Output format: text, json")
}

// analyzeSingleFile contains the full analysis pipeline logic (reused from analyze.go)
func analyzeSingleFile(filePath string) (*analyzer.ComplexityReport, error) {
	data, progSection, err := parser.ParseELF(filePath)
	if err != nil {
		return nil, err
	}

	insts, err := parser.DecodeInstructions(data)
	if err != nil {
		return nil, err
	}

	// Convert to asm.Instruction for analyzer
	asmInsts := parser.ConvertToASM(insts)

	// Build CFG
	blocks := cfg.BuildBasicBlocks(insts)
	progCFG := cfg.BuildCFG(blocks)

	// Use the global profile name if it was set, otherwise default
	profile := "default"
	if profileName != "" {
		profile = profileName
	}

	report, err := analyzer.Analyze(filePath, asmInsts, progCFG, profile)
	if err != nil {
		return nil, err
	}

	report.Section = progSection

	return report, nil
}
