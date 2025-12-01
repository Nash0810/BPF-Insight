package main

import (
	"fmt"
	"github.com/Nash0810/BPF-Insight/pkg/analyzer"
	"github.com/Nash0810/BPF-Insight/pkg/utils"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var batchCmd = &cobra.Command{
	Use:   "batch <directory>",
	Short: "Batch process multiple eBPF programs",
	Long: `Analyzes all eBPF object files in a directory and generates a summary report.
Supports recursive directory traversal and custom file patterns.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		directory := args[0]
		recursive, _ := cmd.Flags().GetBool("recursive")
		pattern, _ := cmd.Flags().GetString("pattern")
		outputFile, _ := cmd.Flags().GetString("output")
		failThreshold, _ := cmd.Flags().GetFloat64("fail-threshold")
		outputFormat, _ := cmd.Flags().GetString("output-format")

		// Find all matching files
		files, err := findFiles(directory, recursive, pattern)
		if err != nil {
			return fmt.Errorf("failed to find files: %w", err)
		}

		if len(files) == 0 {
			return fmt.Errorf("no matching files found in %s (pattern: %s)", directory, pattern)
		}

		// Process each file
		batchReport := &analyzer.BatchReport{
			TotalFiles: len(files),
			Results:    []analyzer.BatchResult{},
			HighRisk:   []string{},
		}

		for _, file := range files {
			result := analyzer.BatchResult{File: file}

			complexityReport, err := analyzeSingleFile(file)
			if err != nil {
				result.Error = err.Error()
				batchReport.Errors++
			} else {
				result.Score = complexityReport.TotalScore
				result.Prediction = complexityReport.Prediction
				batchReport.Analyzed++

				if result.Score > 70 {
					batchReport.HighRisk = append(batchReport.HighRisk, file)
				}
			}

			batchReport.Results = append(batchReport.Results, result)
		}

		// Calculate average score
		totalScore := 0.0
		count := 0
		for _, r := range batchReport.Results {
			if r.Error == "" {
				totalScore += r.Score
				count++
			}
		}
		if count > 0 {
			batchReport.AvgScore = totalScore / float64(count)
		}

		// Output results
		var output *os.File
		if outputFile != "" {
			output, err = os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer output.Close()
		} else {
			output = os.Stdout
		}

		if outputFormat == "json" {
			return utils.PrintJSONToFile(batchReport, output)
		}

		// Text output
		fmt.Fprintf(output, "Batch Analysis Report\n")
		fmt.Fprintf(output, "=====================\n")
		fmt.Fprintf(output, "Directory: %s\n", directory)
		fmt.Fprintf(output, "Files found: %d\n", batchReport.TotalFiles)
		fmt.Fprintf(output, "Analyzed: %d\n", batchReport.Analyzed)
		fmt.Fprintf(output, "Errors: %d\n\n", batchReport.Errors)

		fmt.Fprintf(output, "Results Summary:\n")
		fmt.Fprintf(output, "  Average Score: %.1f / 100\n", batchReport.AvgScore)

		highRisk := 0
		mediumRisk := 0
		lowRisk := 0
		for _, r := range batchReport.Results {
			if r.Error == "" {
				if r.Score > 70 {
					highRisk++
				} else if r.Score >= 40 {
					mediumRisk++
				} else {
					lowRisk++
				}
			}
		}

		fmt.Fprintf(output, "  High Risk (>70): %d files\n", highRisk)
		fmt.Fprintf(output, "  Medium Risk (40-70): %d files\n", mediumRisk)
		fmt.Fprintf(output, "  Low Risk (<40): %d files\n\n", lowRisk)

		if len(batchReport.HighRisk) > 0 {
			fmt.Fprintf(output, "High Risk Files:\n")
			for i, file := range batchReport.HighRisk {
				var result *analyzer.BatchResult
				for _, r := range batchReport.Results {
					if r.File == file {
						result = &r
						break
					}
				}
				if result != nil {
					fmt.Fprintf(output, "  %d. %s - Score: %.1f (%s)\n",
						i+1, filepath.Base(file), result.Score, result.Prediction)
				}
			}
			fmt.Fprintf(output, "\n")
		}

		fmt.Fprintf(output, "Detailed Results:\n")
		for _, r := range batchReport.Results {
			status := "✓"
			if r.Error != "" {
				status = "✗"
			}
			fmt.Fprintf(output, "  %s %-30s", status, filepath.Base(r.File))
			if r.Error == "" {
				fmt.Fprintf(output, " %.1f  %s\n", r.Score, r.Prediction)
			} else {
				fmt.Fprintf(output, " ERROR: %s\n", r.Error)
			}
		}

		// Check fail threshold
		if failThreshold > 0 {
			for _, r := range batchReport.Results {
				if r.Error == "" && r.Score > failThreshold {
					return fmt.Errorf("file %s exceeds fail threshold (%.1f > %.1f)", r.File, r.Score, failThreshold)
				}
			}
		}

		return nil
	},
}

func init() {
	batchCmd.Flags().BoolP("recursive", "r", false, "Process subdirectories")
	batchCmd.Flags().String("pattern", "*.o", "File glob pattern")
	batchCmd.Flags().StringP("output", "o", "", "Output file for results (default: stdout)")
	batchCmd.Flags().Float64("fail-threshold", 0, "Exit with error if any score > threshold")
	batchCmd.Flags().StringP("output-format", "f", "text", "Output format: text, json")
}

// findFiles finds all files matching the pattern in the directory
func findFiles(directory string, recursive bool, pattern string) ([]string, error) {
	var files []string

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		matched, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return err
		}
		if matched {
			files = append(files, path)
		}
		return nil
	}

	if recursive {
		err := filepath.Walk(directory, walkFunc)
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(directory)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				matched, err := filepath.Match(pattern, entry.Name())
				if err != nil {
					return nil, err
				}
				if matched {
					files = append(files, filepath.Join(directory, entry.Name()))
				}
			}
		}
	}

	return files, nil
}
