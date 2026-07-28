// Package app orchestrates a full scan: parse plan → risk → policies.
package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/SamyBaouche/iac-sentinel/internal/policy"
	"github.com/SamyBaouche/iac-sentinel/internal/risk"
	"github.com/SamyBaouche/iac-sentinel/internal/tfplan"
)

// ChangeRisk is one planned change with its classified risk level.
type ChangeRisk struct {
	Address string
	Type    string
	Action  tfplan.Action
	Level   risk.Level
}

// Report is the full scan result shown by the CLI.
type Report struct {
	PlanPath string
	Summary  tfplan.Summary
	Changes  []ChangeRisk
	MaxRisk  risk.Level
	Policy   policy.Result
}

// Options controls how Scan runs.
type Options struct {
	PlanPath     string
	TerraformDir string
	SkipCheckov  bool
	SkipTfsec    bool
	SkipOPA      bool
}

// Run loads the plan, classifies risk, and runs policy scanners.
func Run(ctx context.Context, opts Options) (Report, error) {
	plan, err := tfplan.ParseFile(opts.PlanPath)
	if err != nil {
		return Report{}, err
	}

	summary := tfplan.Summarize(plan)
	changes := make([]ChangeRisk, 0, len(summary.Changes))
	maxRisk := risk.SAFE

	for _, rc := range summary.Changes {
		action := rc.Change.Action()
		level := risk.Classify(action, rc.Type)
		changes = append(changes, ChangeRisk{
			Address: rc.Address,
			Type:    rc.Type,
			Action:  action,
			Level:   level,
		})
		if level > maxRisk {
			maxRisk = level
		}
	}

	pol, err := policy.Scan(ctx, plan, policy.ScanOptions{
		TerraformDir: opts.TerraformDir,
		SkipCheckov:  opts.SkipCheckov,
		SkipTfsec:    opts.SkipTfsec,
		SkipOPA:      opts.SkipOPA,
	})
	if err != nil {
		return Report{}, err
	}

	return Report{
		PlanPath: opts.PlanPath,
		Summary:  summary,
		Changes:  changes,
		MaxRisk:  maxRisk,
		Policy:   pol,
	}, nil
}

// ParseFailOn converts a CLI string into a risk.Level.
// Accepted values: SAFE, CAUTION, DANGER, CRITICAL (case-insensitive).
// Empty string means "do not fail the process based on threshold".
func ParseFailOn(s string) (risk.Level, bool, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return risk.SAFE, false, nil
	}
	switch strings.ToUpper(s) {
	case "SAFE":
		return risk.SAFE, true, nil
	case "CAUTION":
		return risk.CAUTION, true, nil
	case "DANGER":
		return risk.DANGER, true, nil
	case "CRITICAL":
		return risk.CRITICAL, true, nil
	default:
		return 0, false, fmt.Errorf("invalid --fail-on %q (want SAFE, CAUTION, DANGER, or CRITICAL)", s)
	}
}

// FindingLevel maps a policy severity onto the same scale as risk.Level
// so --fail-on can consider both risk and findings.
func FindingLevel(sev policy.Severity) risk.Level {
	switch sev {
	case policy.SeverityCritical:
		return risk.CRITICAL
	case policy.SeverityHigh:
		return risk.DANGER
	case policy.SeverityMedium:
		return risk.CAUTION
	case policy.SeverityLow, policy.SeverityUnknown:
		return risk.SAFE
	default:
		return risk.SAFE
	}
}

// ShouldFail reports whether the process should exit non-zero.
// enabled=false means --fail-on was not set.
func ShouldFail(rep Report, threshold risk.Level, enabled bool) bool {
	if !enabled {
		return false
	}
	if rep.MaxRisk >= threshold {
		return true
	}
	for _, f := range rep.Policy.Findings {
		if FindingLevel(f.Severity) >= threshold {
			return true
		}
	}
	return false
}
