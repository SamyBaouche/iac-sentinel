package tfplan

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrNotAPlan is returned when the JSON has no format_version field.
var ErrNotAPlan = errors.New("tfplan: not a terraform plan (missing format_version)")

// Parse decodes a terraform plan JSON document from r.
func Parse(r io.Reader) (*Plan, error) {
	var plan Plan
	if err := json.NewDecoder(r).Decode(&plan); err != nil {
		return nil, fmt.Errorf("tfplan: decode: %w", err)
	}
	if plan.FormatVersion == "" {
		return nil, ErrNotAPlan
	}
	return &plan, nil
}

// ParseFile opens path and parses it as a terraform plan JSON document.
func ParseFile(path string) (*Plan, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("tfplan: open %s: %w", path, err)
	}
	defer f.Close()

	plan, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("tfplan: parse %s: %w", path, err)
	}
	return plan, nil
}
