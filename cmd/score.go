package main

import (
    "encoding/json"
    "fmt"
    "github.com/spf13/cobra"
    "github.com/yourusername/bpfinsight/pkg/cfg"
    "github.com/yourusername/bpfinsight/pkg/parser"
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
        loops := cfg.DetectLoops(graph)

        score := cfg.ScoreProgram(graph, loops)

        out, _ := json.MarshalIndent(score, "", "  ")
        fmt.Println(string(out))

        return nil
    },
}

func init() {
    rootCmd.AddCommand(scoreCmd)
}
