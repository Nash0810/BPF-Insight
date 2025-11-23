package main

import (
    "fmt"

    "github.com/spf13/cobra"

    "github.com/yourusername/bpfinsight/pkg/parser"
    "github.com/yourusername/bpfinsight/pkg/cfg"
)

var cfgCmd = &cobra.Command{
    Use:   "cfg <file>",
    Short: "Print basic block and CFG summary",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        file := args[0]

        // ELF parser
        p := parser.ELFParser{FilePath: file}
        raw, err := p.Parse()
        if err != nil {
            return fmt.Errorf("ELF parse error: %w", err)
        }

        // Decode
        insns, err := parser.DecodeInstructions(raw)
        if err != nil {
            return fmt.Errorf("decode error: %w", err)
        }

        // Build blocks & CFG
        blocks := cfg.BuildBasicBlocks(insns)
        graph := cfg.BuildCFG(blocks)
        loops := cfg.DetectLoops(graph)

        fmt.Printf("CFG Summary for %s\n", file)
        fmt.Println("=================================")
        fmt.Printf("Basic Blocks: %d\n", len(graph.Blocks))
        fmt.Printf("Loops Detected: %d\n\n", len(loops))

        for _, b := range graph.Blocks {
            fmt.Printf("Block B%d:\n", b.ID)
            fmt.Printf("  Instructions: %d\n", len(b.Instructions))
            fmt.Printf("  Successors: %v\n", b.Successors)
            fmt.Printf("  Predecessors: %v\n\n", b.Predecessors)
        }

        if len(loops) > 0 {
            fmt.Println("Loops:")
            for _, l := range loops {
                fmt.Printf("  Loop header: B%d, blocks: %v\n",
                    l.Header, l.Blocks)
            }
        }

        return nil
    },
}
