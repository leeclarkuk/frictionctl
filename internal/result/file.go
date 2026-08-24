package result

import (
	"encoding/json"
	"fmt"
	"os"
)

// WriteFile writes a result as indented JSON.
func WriteFile(path string, r Result) error {
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write result %s: %w", path, err)
	}
	return nil
}

// LoadFile reads a result JSON file.
func LoadFile(path string) (Result, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read result %s: %w", path, err)
	}
	var r Result
	if err := json.Unmarshal(raw, &r); err != nil {
		return Result{}, fmt.Errorf("parse result %s: %w", path, err)
	}
	if r.Journey == "" {
		return Result{}, fmt.Errorf("result %s: journey is required", path)
	}
	return r, nil
}
