package policy

import (
	"context"

	"github.com/SamyBaouche/tfguard/internal/tfplan"
)

// ScanOptions controls which scanners run.
type ScanOptions struct {
	TerraformDir string // HCL tree for Checkov/tfsec; empty skips both
	SkipCheckov  bool
	SkipTfsec    bool
	SkipOPA      bool
}

// Scan merges Checkov, tfsec, and OPA into one Result.
// Missing optional CLIs only produce warnings.
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
