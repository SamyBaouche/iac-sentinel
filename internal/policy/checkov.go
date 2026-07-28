package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// checkovReport is the subset of Checkov JSON output we care about.
// Checkov prints a large document; we only map failed_checks.
type checkovReport struct {
	FailedChecks []checkovCheck `json:"failed_checks"`
	Results      *struct {
		FailedChecks []checkovCheck `json:"failed_checks"`
	} `json:"results"`
}

type checkovCheck struct {
	CheckID    string `json:"check_id"`
	CheckName  string `json:"check_name"`
	CheckClass string `json:"check_class"`
	Resource   string `json:"resource"`
	FilePath   string `json:"file_path"`
	Guideline  string `json:"guideline"`
	Severity   string `json:"severity"`
}

// RunCheckov executes the Checkov CLI on dir and converts failed checks
// into Finding values.
//
// Behaviour when Checkov is missing:
//   - we do NOT return a hard error
//   - we add a Warning so the caller can print it
//   - Findings stays empty
func RunCheckov(ctx context.Context, dir string) (Result, error) {
	path, err := exec.LookPath("checkov")
	if err != nil {
		return Result{
			Warnings: []string{"checkov not found on PATH; skipping Checkov scan"},
		}, nil
	}

	// checkov -d <dir> -o json --quiet
	// Exit code can be non-zero when findings exist; that is normal.
	cmd := exec.CommandContext(ctx, path, "-d", dir, "-o", "json", "--quiet", "--compact")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	out := stdout.Bytes()
	if len(bytes.TrimSpace(out)) == 0 {
		// No JSON at all — if the binary failed for a real reason, surface it.
		if runErr != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = runErr.Error()
			}
			return Result{}, fmt.Errorf("policy: checkov: %s", msg)
		}
		return Result{}, nil
	}

	findings, err := parseCheckovJSON(out)
	if err != nil {
		return Result{}, fmt.Errorf("policy: checkov parse: %w", err)
	}
	return Result{Findings: findings}, nil
}

// parseCheckovJSON turns Checkov JSON bytes into []Finding.
// Exported for tests via this package-level helper (unexported name).
func parseCheckovJSON(data []byte) ([]Finding, error) {
	var report checkovReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}

	checks := report.FailedChecks
	if len(checks) == 0 && report.Results != nil {
		checks = report.Results.FailedChecks
	}

	findings := make([]Finding, 0, len(checks))
	for _, c := range checks {
		findings = append(findings, Finding{
			ID:          c.CheckID,
			Source:      SourceCheckov,
			Severity:    normalizeSeverity(c.Severity),
			Title:       c.CheckName,
			Description: c.CheckName,
			Resource:    c.Resource,
			File:        c.FilePath,
			Guideline:   c.Guideline,
		})
	}
	return findings, nil
}

func normalizeSeverity(s string) Severity {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CRITICAL":
		return SeverityCritical
	case "HIGH":
		return SeverityHigh
	case "MEDIUM", "MODERATE":
		return SeverityMedium
	case "LOW":
		return SeverityLow
	default:
		return SeverityUnknown
	}
}
