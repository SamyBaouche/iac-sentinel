package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SamyBaouche/tfguard/internal/app"
	"github.com/SamyBaouche/tfguard/internal/policy"
	"github.com/SamyBaouche/tfguard/internal/risk"
	"github.com/SamyBaouche/tfguard/internal/tfplan"
)

func TestTerminal(t *testing.T) {
	t.Parallel()
	rep := app.Report{
		PlanPath: "plan.json",
		MaxRisk:  risk.CRITICAL,
		Summary:  tfplan.Summary{Deletes: 1},
		Changes: []app.ChangeRisk{{
			Address: "aws_db_instance.main",
			Type:    "aws_db_instance",
			Action:  tfplan.ActionDelete,
			Level:   risk.CRITICAL,
		}},
		Policy: policy.Result{
			Findings: []policy.Finding{{
				Severity: policy.SeverityHigh,
				Source:   policy.SourceOPA,
				ID:       "TFGUARD-RDS-001",
				Resource: "aws_db_instance.main",
				Title:    "unencrypted",
			}},
		},
	}
	var buf bytes.Buffer
	if err := Terminal(&buf, rep); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Max risk:", "CRITICAL", "TFGUARD-RDS-001"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q\n%s", want, out)
		}
	}
}
