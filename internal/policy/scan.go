package policy

import (
	"context"

	"github.com/SamyBaouche/iac-sentinel/internal/tfplan"
)

// ScanOptions controls which scanners run.
type ScanOptions struct {
	// TerraformDir is the folder Checkov/tfsec scan (HCL sources).
	// Optional scanners are skipped (with a warning) when empty or when
	// the binary is missing.
	TerraformDir string

	// SkipCheckov / SkipTfsec force-skip even if the binary exists.
	SkipCheckov bool
	SkipTfsec   bool
	SkipOPA     bool
}

// Scan runs optional external scanners + OPA policies and merges Findings.
//
// plan may be nil when you only want Checkov/tfsec on TerraformDir.
// External tools never hard-fail the whole scan solely because they are absent.
func Scan(ctx context.Context, plan *tfplan.Plan, opts ScanOptions) (Result, error) {
	var out Result

	if opts.TerraformDir != "" && !opts.SkipCheckov {
		r, err := RunCheckov(ctx, opts.TerraformDir)
		if err != nil {
			return Result{}, err
		}
		out.Findings = append(out.Findings, r.Findings...)
		out.Warnings = append(out.Warnings, r.Warnings...)
	}

	if opts.TerraformDir != "" && !opts.SkipTfsec {
		r, err := RunTfsec(ctx, opts.TerraformDir)
		if err != nil {
			return Result{}, err
		}
		out.Findings = append(out.Findings, r.Findings...)
		out.Warnings = append(out.Warnings, r.Warnings...)
	}

	if !opts.SkipOPA {
		input, err := PlanInput(plan)
		if err != nil {
			return Result{}, err
		}
		r, err := EvaluateOPA(ctx, input)
		if err != nil {
			return Result{}, err
		}
		out.Findings = append(out.Findings, r.Findings...)
		out.Warnings = append(out.Warnings, r.Warnings...)
	}

	return out, nil
}
