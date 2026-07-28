package policy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SamyBaouche/iac-sentinel/internal/tfplan"
)

func testdata(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestParseCheckovJSON(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(testdata("checkov_failed.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	findings, err := parseCheckovJSON(data)
	if err != nil {
		t.Fatalf("parseCheckovJSON: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("len = %d, want 1", len(findings))
	}
	f := findings[0]
	if f.Source != SourceCheckov {
		t.Fatalf("Source = %q, want checkov", f.Source)
	}
	if f.ID != "CKV_AWS_20" {
		t.Fatalf("ID = %q, want CKV_AWS_20", f.ID)
	}
	if f.Severity != SeverityHigh {
		t.Fatalf("Severity = %q, want HIGH", f.Severity)
	}
	if f.Resource != "aws_s3_bucket.logs" {
		t.Fatalf("Resource = %q", f.Resource)
	}
}

func TestParseTfsecJSON(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(testdata("tfsec_failed.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	findings, err := parseTfsecJSON(data)
	if err != nil {
		t.Fatalf("parseTfsecJSON: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("len = %d, want 1", len(findings))
	}
	f := findings[0]
	if f.Source != SourceTfsec || f.ID != "AVD-AWS-0086" {
		t.Fatalf("unexpected finding: %+v", f)
	}
}

func TestRunCheckovMissingBinary(t *testing.T) {
	// Cannot use t.Parallel with t.Setenv.
	t.Setenv("PATH", "/nonexistent")

	r, err := RunCheckov(context.Background(), ".")
	if err != nil {
		t.Fatalf("missing checkov must not error, got: %v", err)
	}
	if len(r.Findings) != 0 {
		t.Fatalf("Findings = %d, want 0", len(r.Findings))
	}
	if len(r.Warnings) != 1 {
		t.Fatalf("Warnings = %v, want 1 warning", r.Warnings)
	}
}

func TestRunTfsecMissingBinary(t *testing.T) {
	// Cannot use t.Parallel with t.Setenv.
	t.Setenv("PATH", "/nonexistent")

	r, err := RunTfsec(context.Background(), ".")
	if err != nil {
		t.Fatalf("missing tfsec must not error, got: %v", err)
	}
	if len(r.Warnings) != 1 {
		t.Fatalf("Warnings = %v, want 1 warning", r.Warnings)
	}
}

func TestEvaluateOPAPolicies(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"resource_changes": []any{
			map[string]any{
				"address": "aws_s3_bucket.public",
				"type":    "aws_s3_bucket",
				"change": map[string]any{
					"actions": []any{"create"},
					"after":   map[string]any{"acl": "public-read"},
				},
			},
			map[string]any{
				"address": "aws_db_instance.main",
				"type":    "aws_db_instance",
				"change": map[string]any{
					"actions": []any{"create"},
					"after":   map[string]any{"storage_encrypted": false},
				},
			},
			map[string]any{
				"address": "aws_ebs_volume.data",
				"type":    "aws_ebs_volume",
				"change": map[string]any{
					"actions": []any{"create"},
					"after":   map[string]any{"encrypted": false},
				},
			},
			map[string]any{
				"address": "aws_security_group.open",
				"type":    "aws_security_group",
				"change": map[string]any{
					"actions": []any{"create"},
					"after": map[string]any{
						"ingress": []any{
							map[string]any{"cidr_blocks": []any{"0.0.0.0/0"}},
						},
					},
				},
			},
			map[string]any{
				"address": "aws_iam_policy.admin",
				"type":    "aws_iam_policy",
				"change": map[string]any{
					"actions": []any{"create"},
					"after": map[string]any{
						"policy": `{"Statement":[{"Action":"*","Effect":"Allow","Resource":"*"}]}`,
					},
				},
			},
		},
	}

	r, err := EvaluateOPA(context.Background(), input)
	if err != nil {
		t.Fatalf("EvaluateOPA: %v", err)
	}

	wantIDs := map[string]bool{
		"SENTINEL-S3-001":  false,
		"SENTINEL-RDS-001": false,
		"SENTINEL-EBS-001": false,
		"SENTINEL-SG-001":  false,
		"SENTINEL-IAM-001": false,
	}
	for _, f := range r.Findings {
		if f.Source != SourceOPA {
			t.Fatalf("Source = %q, want opa", f.Source)
		}
		if _, ok := wantIDs[f.ID]; ok {
			wantIDs[f.ID] = true
		}
	}
	for id, seen := range wantIDs {
		if !seen {
			t.Fatalf("missing finding %s; got %+v", id, r.Findings)
		}
	}
}

func TestPlanInputAndScanOPAOnly(t *testing.T) {
	t.Parallel()

	plan := &tfplan.Plan{
		FormatVersion: "1.2",
		ResourceChanges: []tfplan.ResourceChange{
			{
				Address: "aws_ebs_volume.data",
				Mode:    "managed",
				Type:    "aws_ebs_volume",
				Name:    "data",
				Change: tfplan.Change{
					Actions: []string{"create"},
					After:   json.RawMessage(`{"encrypted":false,"size":10}`),
				},
			},
		},
	}

	r, err := Scan(context.Background(), plan, ScanOptions{
		SkipCheckov: true,
		SkipTfsec:   true,
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(r.Findings) != 1 || r.Findings[0].ID != "SENTINEL-EBS-001" {
		t.Fatalf("Findings = %+v, want SENTINEL-EBS-001", r.Findings)
	}
}

func TestNormalizeSeverity(t *testing.T) {
	t.Parallel()
	if got := normalizeSeverity("high"); got != SeverityHigh {
		t.Fatalf("got %q", got)
	}
	if got := normalizeSeverity(""); got != SeverityUnknown {
		t.Fatalf("got %q", got)
	}
}
