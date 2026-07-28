// Command tfguard reviews Terraform plans for risk and policy violations.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/SamyBaouche/tfguard/internal/app"
	"github.com/SamyBaouche/tfguard/internal/render"
)

// Version can be overridden at build time via -ldflags.
var Version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printHelp(os.Stderr)
		return 2
	}

	switch args[0] {
	case "version":
		fmt.Printf("tfguard %s\n", Version)
		return 0
	case "scan":
		return cmdScan(args[1:])
	case "help", "-h", "--help":
		printHelp(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printHelp(os.Stderr)
		return 2
	}
}

func printHelp(w *os.File) {
	fmt.Fprintf(w, `tfguard — Terraform plan risk and policy reviewer

Usage:
  tfguard <command> [flags]

Commands:
  scan      Analyze a terraform plan JSON file
  version   Print version
  help      Show this help

Examples:
  tfguard scan -plan plan.json
  tfguard scan -plan plan.json -dir ./infra -fail-on CRITICAL
`)
}

func cmdScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	planPath := fs.String("plan", "", "path to terraform plan JSON (required)")
	tfDir := fs.String("dir", "", "Terraform source dir for Checkov/tfsec (optional)")
	failOn := fs.String("fail-on", "", "exit 1 at this level: SAFE|CAUTION|DANGER|CRITICAL")
	skipCheckov := fs.Bool("skip-checkov", false, "skip Checkov")
	skipTfsec := fs.Bool("skip-tfsec", false, "skip tfsec")
	skipOPA := fs.Bool("skip-opa", false, "skip OPA policies")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *planPath == "" {
		fmt.Fprintln(os.Stderr, "error: -plan is required")
		fs.Usage()
		return 2
	}

	threshold, enabled, err := app.ParseFailOn(*failOn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	rep, err := app.Run(context.Background(), app.Options{
		PlanPath:     *planPath,
		TerraformDir: *tfDir,
		SkipCheckov:  *skipCheckov,
		SkipTfsec:    *skipTfsec,
		SkipOPA:      *skipOPA,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if err := render.Terminal(os.Stdout, rep); err != nil {
		fmt.Fprintf(os.Stderr, "error: render: %v\n", err)
		return 1
	}

	if app.ShouldFail(rep, threshold, enabled) {
		fmt.Fprintf(os.Stderr, "fail-on %s triggered (max risk=%s, findings=%d)\n",
			threshold.String(), rep.MaxRisk.String(), len(rep.Policy.Findings))
		return 1
	}
	return 0
}
