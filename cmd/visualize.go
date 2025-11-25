package main

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/Nash0810/BPF-Insight/pkg/parser"
    "github.com/Nash0810/BPF-Insight/pkg/cfg"
)

var render bool

func init() {
    visualizeCmd.Flags().BoolVarP(&render, "render", "r", false,
        "Render PNG using Graphviz (dot)")
}

var visualizeCmd = &cobra.Command{
    Use:   "visualize <file>",
    Short: "Generate CFG visualization with scoring and risk heatmap",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        file := args[0]

        // Parse ELF
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

        // CFG
        blocks := cfg.BuildBasicBlocks(insns)
        graph := cfg.BuildCFG(blocks)
        loops := cfg.DetectLoops(graph)

        // Score
        score := cfg.ScoreProgram(graph, loops)

        // Ensure folders
        os.MkdirAll("out/dot", 0755)
        os.MkdirAll("out/img", 0755)

        base := filepath.Base(file)
        dotFile := filepath.Join("out/dot", base+".dot")

        // Write scored DOT
        if err := graph.WriteDOTScored(dotFile, loops, score); err != nil {
            return fmt.Errorf("DOT generation failed: %w", err)
        }

        fmt.Printf("DOT file written: %s\n", dotFile)

        // Render PNG
        if render {
            pngFile := filepath.Join("out/img", base+".png")
            cmd := exec.Command("dot", "-Tpng", dotFile, "-o", pngFile)

            if err := cmd.Run(); err != nil {
                return fmt.Errorf("Graphviz rendering failed: %w", err)
            }

            fmt.Printf("PNG rendered: %s\n", pngFile)
        }

        return nil
    },
}

func init() {
    rootCmd.AddCommand(visualizeCmd)
}
