package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bpfva",
	Short: "BPFInsight - eBPF Verifier Complexity Analyzer",
}

func main() {
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(cfgCmd)
	rootCmd.AddCommand(visualizeCmd)
	rootCmd.AddCommand(compareCmd)
	rootCmd.AddCommand(batchCmd)

	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
