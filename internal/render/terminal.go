// Package render formats an app.Report for the terminal.
package render

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/SamyBaouche/tfguard/internal/app"
	"github.com/SamyBaouche/tfguard/internal/ui"
)

// Terminal writes a styled scan report to w.
func Terminal(w io.Writer, rep app.Report) error {
	style := ui.NewStyle(w)

	fmt.Fprintln(w)
	ui.BoxTitle(w, style, "Scan report")
	ui.BoxLine(w, style, style.Dim("plan")+"   "+rep.PlanPath)
	ui.BoxLine(w, style, style.Dim("risk")+"   "+style.Risk(rep.MaxRisk.String())+style.Dim("  (highest)"))
	ui.BoxEnd(w, style)
	fmt.Fprintln(w)

	writeSummary(w, style, rep)
	fmt.Fprintln(w)
	if err := writeChanges(w, style, rep); err != nil {
		return err
	}
	fmt.Fprintln(w)
	if err := writeFindings(w, style, rep); err != nil {
		return err
	}
	if len(rep.Policy.Warnings) > 0 {
		fmt.Fprintln(w)
		writeWarnings(w, style, rep)
	}
	fmt.Fprintln(w)
	writeFooter(w, style, rep)
	return nil
}

func writeSummary(w io.Writer, style ui.Style, rep app.Report) {
	ui.BoxTitle(w, style, "Summary")
	line := fmt.Sprintf("%s %s   %s %s   %s %s   %s %s",
		style.Green("create"), style.Bold(itoa(rep.Summary.Creates)),
		style.Yellow("update"), style.Bold(itoa(rep.Summary.Updates)),
		style.Red("replace"), style.Bold(itoa(rep.Summary.Replaces)),
		style.Magenta("delete"), style.Bold(itoa(rep.Summary.Deletes)),
	)
	ui.BoxLine(w, style, line)
	ui.BoxEnd(w, style)
}

func writeChanges(w io.Writer, style ui.Style, rep app.Report) error {
	ui.Section(w, style, "Changes")
	if len(rep.Changes) == 0 {
		fmt.Fprintln(w, style.Dim("  (none)"))
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
		style.Dim("RISK"), style.Dim("ACTION"), style.Dim("TYPE"), style.Dim("ADDRESS"))
	for _, c := range rep.Changes {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
			style.Risk(c.Level.String()),
			c.Action,
			style.Dim(c.Type),
			c.Address,
		)
	}
	return tw.Flush()
}

func writeFindings(w io.Writer, style ui.Style, rep app.Report) error {
	ui.Section(w, style, "Policy findings")
	if len(rep.Policy.Findings) == 0 {
		fmt.Fprintln(w, "  "+style.Green("✓")+" "+style.Dim("no findings"))
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n",
		style.Dim("SEV"), style.Dim("SOURCE"), style.Dim("ID"),
		style.Dim("RESOURCE"), style.Dim("TITLE"))
	for _, f := range rep.Policy.Findings {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n",
			style.Risk(string(f.Severity)),
			style.Cyan(string(f.Source)),
			f.ID,
			style.Dim(f.Resource),
			f.Title,
		)
	}
	return tw.Flush()
}

func writeWarnings(w io.Writer, style ui.Style, rep app.Report) {
	ui.Section(w, style, "Warnings")
	for _, warn := range rep.Policy.Warnings {
		fmt.Fprintf(w, "  %s %s\n", style.Yellow("!"), warn)
	}
}

func writeFooter(w io.Writer, style ui.Style, rep app.Report) {
	total := rep.Summary.Creates + rep.Summary.Updates + rep.Summary.Replaces + rep.Summary.Deletes
	msg := fmt.Sprintf("  %s  %d changes scanned · %d policy findings · max risk %s",
		style.Cyan("▸"),
		total,
		len(rep.Policy.Findings),
		style.Risk(rep.MaxRisk.String()),
	)
	fmt.Fprintln(w, msg)
	fmt.Fprintln(w, style.Dim("  "+strings.Repeat("─", 48)))
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
