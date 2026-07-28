package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SamyBaouche/tfguard/internal/policy"
	"github.com/SamyBaouche/tfguard/internal/risk"
)

func TestParseFailOn(t *testing.T) {
	t.Parallel()
	_, enabled, err := ParseFailOn("")
	if err != nil || enabled {
		t.Fatal("empty fail-on should be disabled")
	}
	got, enabled, err := ParseFailOn("danger")
	if err != nil || !enabled || got != risk.DANGER {
		t.Fatalf("got %v enabled=%v err=%v", got, enabled, err)
	}
	if _, _, err := ParseFailOn("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestShouldFail(t *testing.T) {
	t.Parallel()
	rep := Report{MaxRisk: risk.DANGER}
	if ShouldFail(rep, risk.CRITICAL, true) || !ShouldFail(rep, risk.DANGER, true) {
		t.Fatal("unexpected ShouldFail for MaxRisk")
	}
	rep = Report{
		MaxRisk: risk.SAFE,
		Policy:  policy.Result{Findings: []policy.Finding{{Severity: policy.SeverityCritical}}},
	}
	if !ShouldFail(rep, risk.CRITICAL, true) {
		t.Fatal("CRITICAL finding should fail")
	}
}

func TestRunMixedPlan(t *testing.T) {
	t.Parallel()
	rep, err := Run(context.Background(), Options{
		PlanPath:    filepath.Join("..", "..", "testdata", "plan_mixed.json"),
		SkipCheckov: true,
		SkipTfsec:   true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Summary.Creates != 1 || rep.MaxRisk < risk.DANGER {
		t.Fatalf("unexpected report: %+v", rep)
	}
}
