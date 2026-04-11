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

// PrintJSON writes pretty-indented JSON to stdout.
func PrintJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// PrintJSONCompact writes a single-line JSON object to stdout followed by
// a newline. This is the shape the meow-openspec formula's plan step
// captures — the CompileSummary contract mandates "exactly one line that
// parses as a JSON object".
func PrintJSONCompact(data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(b))
	return err
}
