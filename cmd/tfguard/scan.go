package main

import (
	"context"
	"fmt"

	"github.com/SamyBaouche/tfguard/internal/app"
	"github.com/SamyBaouche/tfguard/internal/render"
	"github.com/spf13/cobra"
)

// scanFlags holds CLI options for the scan command.
type scanFlags struct {
	plan        string
	dir         string
	failOn      string
	skipCheckov bool
	skipTfsec   bool
	skipOPA     bool
}

func newScanCmd() *cobra.Command {
	f := &scanFlags{}

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Analyze a terraform plan JSON file",
		Long: `Parse a terraform plan JSON, classify risk for each change,
run policy scanners (OPA + optional Checkov/tfsec), and print a report.

Use --fail-on to exit 1 when risk or findings reach a threshold (CI gate).`,
		Example: `  tfguard scan --plan plan.json
  tfguard scan --plan plan.json --dir ./infra --fail-on CRITICAL
  tfguard scan --plan plan.json --skip-checkov --skip-tfsec`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runScan(cmd, f)
		},
	}

	cmd.Flags().StringVar(&f.plan, "plan", "", "path to terraform plan JSON (required)")
	cmd.Flags().StringVar(&f.dir, "dir", "", "Terraform source directory for Checkov/tfsec")
	cmd.Flags().StringVar(&f.failOn, "fail-on", "", "exit 1 at this level: SAFE|CAUTION|DANGER|CRITICAL")
	cmd.Flags().BoolVar(&f.skipCheckov, "skip-checkov", false, "do not run Checkov")
	cmd.Flags().BoolVar(&f.skipTfsec, "skip-tfsec", false, "do not run tfsec")
	cmd.Flags().BoolVar(&f.skipOPA, "skip-opa", false, "do not run OPA policies")

	_ = cmd.MarkFlagRequired("plan")
	return cmd
}

func runScan(cmd *cobra.Command, f *scanFlags) error {
	threshold, enabled, err := app.ParseFailOn(f.failOn)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	rep, err := app.Run(context.Background(), app.Options{
		PlanPath:     f.plan,
		TerraformDir: f.dir,
		SkipCheckov:  f.skipCheckov,
		SkipTfsec:    f.skipTfsec,
		SkipOPA:      f.skipOPA,
	})
	if err != nil {
		return &exitError{code: 1, msg: err.Error()}
	}

	if err := render.Terminal(cmd.OutOrStdout(), rep); err != nil {
		return &exitError{code: 1, msg: fmt.Sprintf("render: %v", err)}
	}

	if app.ShouldFail(rep, threshold, enabled) {
		return &exitError{
			code: 1,
			msg: fmt.Sprintf("fail-on %s triggered (max risk=%s, findings=%d)",
				threshold.String(), rep.MaxRisk.String(), len(rep.Policy.Findings)),
		}
	}
	return nil
}
