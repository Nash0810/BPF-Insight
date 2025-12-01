// File: pkg/utils/json.go

package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON marshals and prints a struct to stdout in pretty-printed JSON format.
func PrintJSON(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

// MarshalJSON marshals a struct to JSON bytes.
func MarshalJSON(data interface{}) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

// PrintJSONToFile prints any value as formatted JSON to the specified file
func PrintJSONToFile(v interface{}, f *os.File) error {
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
