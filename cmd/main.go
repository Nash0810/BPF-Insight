package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "bpfva",
    Short: "BPFInsight - eBPF Verifier Complexity Analyzer",
    Long:  "Static analysis tool to predict eBPF verifier complexity and failure likelihood.",
}

func main() {
    // Register subcommands
    rootCmd.AddCommand(analyzeCmd)

    // Execute
    if err := rootCmd.Execute(); err != nil {
        panic(err)
    }
}
