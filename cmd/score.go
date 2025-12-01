package main

import (
    "encoding/json"
    "fmt"
    "github.com/spf13/cobra"
    "github.com/Nash0810/BPF-Insight/pkg/cfg"
    "github.com/Nash0810/BPF-Insight/pkg/parser"
)

var scoreCmd = &cobra.Command{
    Use:   "score <file>",
    Short: "Compute block and program complexity scores",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {

        file := args[0]

        p := parser.ELFParser{FilePath: file}
        raw, err := p.Parse()
        if err != nil {
            return fmt.Errorf("ELF parse error: %w", err)
        }

        insns, err := parser.DecodeInstructions(raw)
        if err != nil {
            return fmt.Errorf("decode error: %w", err)
        }

        blocks := cfg.BuildBasicBlocks(insns)
        graph := cfg.BuildCFG(blocks)
        // loops := cfg.DetectLoops(graph) // unused

        // Use CalculateScores for scoring
        // Note: parser.DecodeInstructions returns []parser.Instruction, but CalculateScores expects []asm.Instruction
        // If needed, convert or adapt here. For now, just print block count and basic metrics.
        metrics, hotspots := cfg.CalculateScores(graph, nil)
        out, _ := json.MarshalIndent(struct {
            Metrics  cfg.CFGMetrics
            Hotspots []cfg.BlockComplexity
        }{
            Metrics: metrics,
            Hotspots: hotspots,
        }, "", "  ")
        fmt.Println(string(out))
        return nil
    },
}

func init() {
    rootCmd.AddCommand(scoreCmd)
}
