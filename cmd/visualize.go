package main

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"
    "github.com/Nash0810/BPF-Insight/pkg/parser"
    "github.com/Nash0810/BPF-Insight/pkg/cfg"
    "github.com/Nash0810/BPF-Insight/pkg/analyzer"
)

var (
    renderFlag bool
    outputFile string
    formatFlag string
)

func init() {
    visualizeCmd.Flags().BoolVarP(&renderFlag, "render", "r", false,
        "Auto-render with Graphviz")
    visualizeCmd.Flags().StringVarP(&outputFile, "output", "o", "cfg.dot",
        "Output file path")
    visualizeCmd.Flags().StringVar(&formatFlag, "format", "dot",
        "Output format: dot, png, svg")
}

var visualizeCmd = &cobra.Command{
    Use:   "visualize <file>",
    Short: "Generate CFG visualization with scoring and risk heatmap",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        file := args[0]

        // Use analyzeSingleFile to get full analysis
        report, err := analyzeSingleFile(file)
        if err != nil {
            return err
        }

        if report.CFG == nil {
            return fmt.Errorf("CFG not available")
        }

        fmt.Println("Generating CFG visualization...")
        fmt.Printf("  └─ Parsed %d instructions\n", report.InstructionCount)
        fmt.Printf("  └─ Built CFG with %d basic blocks\n", report.BasicBlockCount)
        fmt.Printf("  └─ Detected %d loops\n", report.LoopCount)

        // Create hotspot map for visualization
        scoreMap := make(cfg.HotspotMap)
        for _, hs := range report.Hotspots {
            scoreMap[hs.BlockID] = hs.Score
        }

        // Generate DOT content
        dotContent := report.CFG.GenerateDOT(scoreMap)

        // Determine output file based on format
        outputPath := outputFile
        if formatFlag != "dot" && !strings.HasSuffix(outputPath, "."+formatFlag) {
            // Replace extension if format is specified
            ext := filepath.Ext(outputPath)
            if ext == "" {
                outputPath = outputPath + "." + formatFlag
            } else {
                outputPath = strings.TrimSuffix(outputPath, ext) + "." + formatFlag
            }
        }

        // If format is dot or render is requested, generate DOT first
        dotFile := outputPath
        if formatFlag != "dot" {
            dotFile = strings.TrimSuffix(outputPath, "."+formatFlag) + ".dot"
        }

        // Write DOT file
        if err := os.WriteFile(dotFile, []byte(dotContent), 0644); err != nil {
            return fmt.Errorf("failed to write DOT file: %w", err)
        }

        fmt.Printf("  └─ Generated DOT file: %s\n", dotFile)

        // Render if requested
        if renderFlag || formatFlag != "dot" {
            fmt.Printf("  └─ Rendering to %s...\n", strings.ToUpper(formatFlag))

            formatMap := map[string]string{
                "png": "png",
                "svg": "svg",
                "pdf": "pdf",
            }

            graphvizFormat, ok := formatMap[formatFlag]
            if !ok {
                graphvizFormat = "png" // default
            }

            renderCmd := exec.Command("dot", "-T"+graphvizFormat, dotFile, "-o", outputPath)
            if err := renderCmd.Run(); err != nil {
                return fmt.Errorf("Graphviz rendering failed (is 'dot' installed?): %w", err)
            }

            fmt.Printf("  └─ Saved: %s\n", outputPath)
        }

        fmt.Println("\nVisualization complete!")
        return nil
    },
}

func init() {
    rootCmd.AddCommand(visualizeCmd)
}
