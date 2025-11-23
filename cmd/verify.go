package main

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/yourusername/bpfinsight/pkg/verify"
    "github.com/yourusername/bpfinsight/pkg/parser"
    "github.com/yourusername/bpfinsight/pkg/cfg"
)

var profile string
var enableRules string
var disableRules string

var verifyCmd = &cobra.Command{
    Use:   "verify <file>",
    Short: "Run eBPF verifier rule analysis",
    Args:  cobra.ExactArgs(1),

    RunE: func(cmd *cobra.Command, args []string) error {
        file := args[0]

        // Apply verification profile + custom rule toggles
        verify.ApplyProfile(profile)
        verify.EnableRules(enableRules)
        verify.DisableRules(disableRules)

        // Parse ELF
        p := parser.ELFParser{FilePath: file}
        raw, err := p.Parse()
        if err != nil {
            return fmt.Errorf("ELF parse error: %w", err)
        }

        insns, err := parser.Decode(raw)
        if err != nil {
            return fmt.Errorf("decode error: %w", err)
        }

        blocks := cfg.BuildBasicBlocks(insns)
        graph := cfg.BuildCFG(blocks)

        result := verify.RunVerifier(insns, graph)

        // Output JSON
        jsonOut, _ := verify.ToJSON(result)
        fmt.Println(string(jsonOut))

        return nil
    },
}

func init() {
    verifyCmd.Flags().StringVar(
        &profile, "profile", "default",
        "Verification profile (default|strict|kernel|relaxed)",
    )
    verifyCmd.Flags().StringVar(
        &enableRules, "rules", "",
        "Enable specific rules (comma-separated rule IDs)",
    )
    verifyCmd.Flags().StringVar(
        &disableRules, "disable-rules", "",
        "Disable specific rules (comma-separated rule IDs)",
    )
}
