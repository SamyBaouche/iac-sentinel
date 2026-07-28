package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SamyBaouche/iac-sentinel/internal/policy"
	"github.com/SamyBaouche/iac-sentinel/internal/risk"
)

func TestParseFailOn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in      string
		want    risk.Level
		enabled bool
		err     bool
	}{
		{"", risk.SAFE, false, false},
		{"CRITICAL", risk.CRITICAL, true, false},
		{"danger", risk.DANGER, true, false},
		{"nope", 0, false, true},
	}

	for _, tt := range tests {
		got, enabled, err := ParseFailOn(tt.in)
		if tt.err {
			if err == nil {
				t.Fatalf("ParseFailOn(%q) expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseFailOn(%q): %v", tt.in, err)
		}
		if got != tt.want || enabled != tt.enabled {
			t.Fatalf("ParseFailOn(%q) = %v,%v want %v,%v", tt.in, got, enabled, tt.want, tt.enabled)
		}
	}
}

func TestShouldFail(t *testing.T) {
	t.Parallel()

	rep := Report{
		MaxRisk: risk.DANGER,
		Policy: policy.Result{
			Findings: []policy.Finding{
				{Severity: policy.SeverityHigh, ID: "X"},
			},
		},
	}

	if ShouldFail(rep, risk.CRITICAL, true) {
		t.Fatal("DANGER should not fail on CRITICAL threshold")
	}
	if !ShouldFail(rep, risk.DANGER, true) {
		t.Fatal("DANGER should fail on DANGER threshold")
	}
	if ShouldFail(rep, risk.CRITICAL, false) {
		t.Fatal("disabled fail-on must never fail")
	}

	repCriticalFinding := Report{
		MaxRisk: risk.SAFE,
		Policy: policy.Result{
			Findings: []policy.Finding{{Severity: policy.SeverityCritical}},
		},
	}
	if !ShouldFail(repCriticalFinding, risk.CRITICAL, true) {
		t.Fatal("CRITICAL finding should fail on CRITICAL threshold")
	}
}

func TestFindingLevel(t *testing.T) {
	t.Parallel()
	if FindingLevel(policy.SeverityHigh) != risk.DANGER {
		t.Fatal("HIGH should map to DANGER")
	}
	if FindingLevel(policy.SeverityCritical) != risk.CRITICAL {
		t.Fatal("CRITICAL should map to CRITICAL")
	}
}

func TestRunMixedPlan(t *testing.T) {
	t.Parallel()

	plan := filepath.Join("..", "..", "testdata", "plan_mixed.json")
	rep, err := Run(context.Background(), Options{
		PlanPath:    plan,
		SkipCheckov: true,
		SkipTfsec:   true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Summary.Creates != 1 || rep.Summary.Deletes != 1 {
		t.Fatalf("unexpected summary: %+v", rep.Summary)
	}
	if rep.MaxRisk < risk.DANGER {
		t.Fatalf("MaxRisk = %s, want at least DANGER (delete present)", rep.MaxRisk)
	}
}
