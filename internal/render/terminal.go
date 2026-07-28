// Package render prints scan reports to the terminal.
package render

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/SamyBaouche/iac-sentinel/internal/app"
)

// Terminal writes a human-readable report to w.
func Terminal(w io.Writer, rep app.Report) error {
	if _, err := fmt.Fprintf(w, "IaC Sentinel — scan report\n"); err != nil {
		return err
	}
	fmt.Fprintf(w, "Plan: %s\n", rep.PlanPath)
	fmt.Fprintf(w, "Max risk: %s\n\n", rep.MaxRisk.String())

	fmt.Fprintln(w, "Summary")
	fmt.Fprintln(w, strings.Repeat("-", 40))
	fmt.Fprintf(w, "  create : %d\n", rep.Summary.Creates)
	fmt.Fprintf(w, "  update : %d\n", rep.Summary.Updates)
	fmt.Fprintf(w, "  replace: %d\n", rep.Summary.Replaces)
	fmt.Fprintf(w, "  delete : %d\n\n", rep.Summary.Deletes)

	fmt.Fprintln(w, "Changes")
	fmt.Fprintln(w, strings.Repeat("-", 40))
	if len(rep.Changes) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "RISK\tACTION\tTYPE\tADDRESS")
		for _, c := range rep.Changes {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", c.Level.String(), c.Action, c.Type, c.Address)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Policy findings")
	fmt.Fprintln(w, strings.Repeat("-", 40))
	if len(rep.Policy.Findings) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "SEV\tSOURCE\tID\tRESOURCE\tTITLE")
		for _, f := range rep.Policy.Findings {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", f.Severity, f.Source, f.ID, f.Resource, f.Title)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	fmt.Fprintln(w)

	if len(rep.Policy.Warnings) > 0 {
		fmt.Fprintln(w, "Warnings")
		fmt.Fprintln(w, strings.Repeat("-", 40))
		for _, warn := range rep.Policy.Warnings {
			fmt.Fprintf(w, "  - %s\n", warn)
		}
		fmt.Fprintln(w)
	}

	return nil
}
