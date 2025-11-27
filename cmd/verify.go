package main

import (
    "encoding/json"
    "fmt"

    "github.com/spf13/cobra"

    "github.com/Nash0810/BPF-Insight/pkg/cfg"
    "github.com/Nash0810/BPF-Insight/pkg/parser"
    "github.com/Nash0810/BPF-Insight/pkg/verify"
)

var (
    flagProfile string
    flagEnable  []string
    flagDisable []string
    flagJSON    bool
)

func init() {
    verifyCmd.Flags().StringVar(&flagProfile, "profile", "default",
        "Verification profile (safe, strict, default, permissive)")

    verifyCmd.Flags().StringSliceVar(&flagEnable, "enable", nil,
        "Enable specific rules (comma-separated)")
    verifyCmd.Flags().StringSliceVar(&flagDisable, "disable", nil,
        "Disable specific rules (comma-separated)")

    verifyCmd.Flags().BoolVar(&flagJSON, "json", false,
        "Output verification result in JSON format")
}

var verifyCmd = &cobra.Command{
    Use:   "verify <file>",
    Short: "Run static verifier rule engine",
    Args:  cobra.ExactArgs(1),

    RunE: func(cmd *cobra.Command, args []string) error {
        file := args[0]

        // ===== 1. Parse ELF =====
        p := parser.ELFParser{FilePath: file}
        raw, err := p.Parse()
        if err != nil {
            return fmt.Errorf("ELF parse error: %w", err)
        }

        // ===== 2. Decode instructions =====
        insns, err := parser.DecodeInstructions(raw)
        if err != nil {
            return fmt.Errorf("decode error: %w", err)
        }

		// ===== 3. Build CFG =====
		blocks := cfg.BuildBasicBlocks(insns)

		// Build full graph for loop detection
		graph := cfg.BuildCFG(blocks)
		loops := cfg.DetectLoops(graph)

		// Convert []*BasicBlock → []BasicBlock
		flatBlocks := make([]cfg.BasicBlock, len(blocks))
		for i, b := range blocks {
			flatBlocks[i] = *b
		}


        // ===== 4. Apply Profile & Manual Flags =====
        if err := verify.ApplyProfile(flagProfile); err != nil {
            return err
        }
		// Apply manual rule enable/disable overrides (Step-3)
		for _, r := range flagEnable {
			if rule, ok := verify.Rules[r]; ok {
				rule.Enabled = true
			}
		}

		for _, r := range flagDisable {
			if rule, ok := verify.Rules[r]; ok {
				rule.Enabled = false
			}
		}


        // ===== 5. Run Verifier =====
        result := verify.VerifyProgram(flatBlocks, loops)

        // ===== 6. Output =====
        if flagJSON {
            out, _ := json.MarshalIndent(result, "", "  ")
            fmt.Println(string(out))
            return nil
        }

        // Human readable
        fmt.Println("Verification Result:")
        fmt.Println("─────────────────────")

        for _, w := range result.BlockWarnings {
            fmt.Printf("[Block %d] %s\n", w.BlockID, w.Message)
        }
        for _, w := range result.ProgramWarnings {
            fmt.Printf("[Program] %s\n", w)
        }

        fmt.Println()
        fmt.Printf("Prediction: %s\n", result.FinalPrediction)

        return nil
    },
}

func init() {
    rootCmd.AddCommand(verifyCmd)
}
