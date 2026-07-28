package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SamyBaouche/iac-sentinel/internal/app"
	"github.com/SamyBaouche/iac-sentinel/internal/policy"
	"github.com/SamyBaouche/iac-sentinel/internal/risk"
	"github.com/SamyBaouche/iac-sentinel/internal/tfplan"
)

func TestTerminal(t *testing.T) {
	t.Parallel()

	rep := app.Report{
		PlanPath: "testdata/plan.json",
		MaxRisk:  risk.CRITICAL,
		Summary:  tfplan.Summary{Creates: 1, Deletes: 1},
		Changes: []app.ChangeRisk{
			{Address: "aws_db_instance.main", Type: "aws_db_instance", Action: tfplan.ActionDelete, Level: risk.CRITICAL},
		},
		Policy: policy.Result{
			Findings: []policy.Finding{
				{Severity: policy.SeverityHigh, Source: policy.SourceOPA, ID: "SENTINEL-RDS-001", Resource: "aws_db_instance.main", Title: "unencrypted"},
			},
			Warnings: []string{"checkov not found on PATH; skipping Checkov scan"},
		},
	}

	var buf bytes.Buffer
	if err := Terminal(&buf, rep); err != nil {
		t.Fatalf("Terminal: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Summary", "Changes", "Policy findings", "Warnings", "CRITICAL", "SENTINEL-RDS-001"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output:\n%s", want, out)
		}
	}
}
