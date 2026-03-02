package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintOutput writes either JSON or plain text depending on the --json flag.
func PrintOutput(data any, text string) {
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(data)
	} else {
		fmt.Println(text)
	}
}

// PrintJSON writes JSON to stdout.
func PrintJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
