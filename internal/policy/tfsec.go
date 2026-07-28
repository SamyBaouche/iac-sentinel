package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Subset of tfsec / aquasec JSON output.
type tfsecReport struct {
	Results []tfsecResult `json:"results"`
}

type tfsecResult struct {
	RuleID      string `json:"rule_id"`
	LongID      string `json:"long_id"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Resource    string `json:"resource"`
	Link        string `json:"link"`
	Location    struct {
		Filename string `json:"filename"`
	} `json:"location"`
}

// RunTfsec executes tfsec on dir and converts results to Findings.
// If the binary is missing, returns a Warning and a nil error.
func RunTfsec(ctx context.Context, dir string) (Result, error) {
	path, err := exec.LookPath("tfsec")
	if err != nil {
		return Result{Warnings: []string{"tfsec not found on PATH; skipping tfsec scan"}}, nil
	}

	cmd := exec.CommandContext(ctx, path, dir, "--format", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	out := stdout.Bytes()
	if len(bytes.TrimSpace(out)) == 0 {
		if runErr != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = runErr.Error()
			}
			return Result{}, fmt.Errorf("policy: tfsec: %s", msg)
		}
		return Result{}, nil
	}

	findings, err := parseTfsecJSON(out)
	if err != nil {
		return Result{}, fmt.Errorf("policy: tfsec parse: %w", err)
	}
	return Result{Findings: findings}, nil
}

func parseTfsecJSON(data []byte) ([]Finding, error) {
	var report tfsecReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}

	findings := make([]Finding, 0, len(report.Results))
	for _, r := range report.Results {
		id := r.RuleID
		if id == "" {
			id = r.LongID
		}
		findings = append(findings, Finding{
			ID:          id,
			Source:      SourceTfsec,
			Severity:    normalizeSeverity(r.Severity),
			Title:       r.Description,
			Description: r.Description,
			Resource:    r.Resource,
			File:        r.Location.Filename,
			Guideline:   r.Link,
		})
	}
	return findings, nil
}
