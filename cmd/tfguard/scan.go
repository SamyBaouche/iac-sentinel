package main

import (
	"context"
	"fmt"

	"github.com/SamyBaouche/tfguard/internal/app"
	"github.com/SamyBaouche/tfguard/internal/render"
	"github.com/SamyBaouche/tfguard/internal/ui"
	"github.com/spf13/cobra"
)

// scanFlags holds CLI options for the scan command.
type scanFlags struct {
	plan             string
	dir              string
	failOn           string
	maxCostIncrease  string
	skipCheckov      bool
	skipTfsec        bool
	skipOPA          bool
	skipCost         bool
	noBanner         bool
}

func newScanCmd() *cobra.Command {
	f := &scanFlags{}

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Analyze a terraform plan JSON file",
		Long: `Parse a terraform plan JSON, classify risk for each change,
run policy scanners (OPA + optional Checkov/tfsec), estimate a static
AWS monthly cost delta, and print a report.

Use --fail-on to exit 1 when risk or findings reach a threshold (CI gate).
Use --max-cost-increase to exit 1 when the monthly cost delta exceeds a USD ceiling.

Progress steps animate in a TTY; set NO_COLOR=1 for plain output.`,
		Example: `  tfguard scan --plan plan.json
  tfguard scan --plan plan.json --dir ./infra --fail-on CRITICAL
  tfguard scan --plan plan.json --max-cost-increase 50
  tfguard scan --plan plan.json --skip-checkov --skip-tfsec`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runScan(cmd, f)
		},
	}

	cmd.Flags().StringVar(&f.plan, "plan", "", "path to terraform plan JSON (required)")
	cmd.Flags().StringVar(&f.dir, "dir", "", "Terraform source directory for Checkov/tfsec")
	cmd.Flags().StringVar(&f.failOn, "fail-on", "", "exit 1 at this level: SAFE|CAUTION|DANGER|CRITICAL")
	cmd.Flags().StringVar(&f.maxCostIncrease, "max-cost-increase", "", "exit 1 if monthly cost delta USD exceeds this value")
	cmd.Flags().BoolVar(&f.skipCheckov, "skip-checkov", false, "do not run Checkov")
	cmd.Flags().BoolVar(&f.skipTfsec, "skip-tfsec", false, "do not run tfsec")
	cmd.Flags().BoolVar(&f.skipOPA, "skip-opa", false, "do not run OPA policies")
	cmd.Flags().BoolVar(&f.skipCost, "skip-cost", false, "do not estimate AWS cost delta")
	cmd.Flags().BoolVar(&f.noBanner, "no-banner", false, "skip the animated banner")

	_ = cmd.MarkFlagRequired("plan")
	return cmd
}

func runScan(cmd *cobra.Command, f *scanFlags) error {
	threshold, failOnEnabled, err := app.ParseFailOn(f.failOn)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	costLimit, costEnabled, err := app.ParseMaxCostIncrease(f.maxCostIncrease)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	out := cmd.OutOrStdout()
	errW := cmd.ErrOrStderr()
	style := ui.NewStyle(errW)

	if !f.noBanner {
		ui.Banner(errW, style, Version)
	}

	spin := ui.NewSpinner(errW, style)

	rep, err := app.Run(context.Background(), app.Options{
		PlanPath:     f.plan,
		TerraformDir: f.dir,
		SkipCheckov:  f.skipCheckov,
		SkipTfsec:    f.skipTfsec,
		SkipOPA:      f.skipOPA,
		SkipCost:     f.skipCost,
		Progress:     spin,
	})
	if err != nil {
		return &exitError{code: 1, msg: err.Error()}
	}

	if err := render.Terminal(out, rep); err != nil {
		return &exitError{code: 1, msg: fmt.Sprintf("render: %v", err)}
	}

	if app.ShouldFail(rep, threshold, failOnEnabled) {
		return &exitError{
			code: 1,
			msg: fmt.Sprintf("fail-on %s triggered (max risk=%s, findings=%d)",
				threshold.String(), rep.MaxRisk.String(), len(rep.Policy.Findings)),
		}
	}
	if app.CostExceeded(rep, costLimit, costEnabled) {
		return &exitError{
			code: 1,
			msg: fmt.Sprintf("max-cost-increase %.2f triggered (delta=%+.2f USD/mo)",
				costLimit, rep.Cost.MonthlyDeltaUSD),
		}
	}
	return nil
}
