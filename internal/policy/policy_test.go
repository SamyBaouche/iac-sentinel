package policy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SamyBaouche/tfguard/internal/tfplan"
)

func testdata(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestParseCheckovJSON(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(testdata("checkov_failed.json"))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := parseCheckovJSON(data)
	if err != nil || len(findings) != 1 || findings[0].ID != "CKV_AWS_20" {
		t.Fatalf("got %+v err=%v", findings, err)
	}
}

func TestParseTfsecJSON(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(testdata("tfsec_failed.json"))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := parseTfsecJSON(data)
	if err != nil || len(findings) != 1 || findings[0].ID != "AVD-AWS-0086" {
		t.Fatalf("got %+v err=%v", findings, err)
	}
}

func TestRunCheckovMissingBinary(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")
	r, err := RunCheckov(context.Background(), ".")
	if err != nil || len(r.Warnings) != 1 {
		t.Fatalf("want warning, got %+v err=%v", r, err)
	}
}

func TestRunTfsecMissingBinary(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")
	r, err := RunTfsec(context.Background(), ".")
	if err != nil || len(r.Warnings) != 1 {
		t.Fatalf("want warning, got %+v err=%v", r, err)
	}
}

func TestEvaluateOPAPolicies(t *testing.T) {
	t.Parallel()
	input := map[string]any{
		"resource_changes": []any{
			map[string]any{
				"address": "aws_s3_bucket.public",
				"type":    "aws_s3_bucket",
				"change":  map[string]any{"actions": []any{"create"}, "after": map[string]any{"acl": "public-read"}},
			},
			map[string]any{
				"address": "aws_db_instance.main",
				"type":    "aws_db_instance",
				"change":  map[string]any{"actions": []any{"create"}, "after": map[string]any{"storage_encrypted": false}},
			},
			map[string]any{
				"address": "aws_ebs_volume.data",
				"type":    "aws_ebs_volume",
				"change":  map[string]any{"actions": []any{"create"}, "after": map[string]any{"encrypted": false}},
			},
			map[string]any{
				"address": "aws_security_group.open",
				"type":    "aws_security_group",
				"change": map[string]any{
					"actions": []any{"create"},
					"after":   map[string]any{"ingress": []any{map[string]any{"cidr_blocks": []any{"0.0.0.0/0"}}}},
				},
			},
			map[string]any{
				"address": "aws_iam_policy.admin",
				"type":    "aws_iam_policy",
				"change": map[string]any{
					"actions": []any{"create"},
					"after":   map[string]any{"policy": `{"Statement":[{"Action":"*","Effect":"Allow","Resource":"*"}]}`},
				},
			},
		},
	}

	r, err := EvaluateOPA(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"TFGUARD-S3-001": false, "TFGUARD-RDS-001": false, "TFGUARD-EBS-001": false,
		"TFGUARD-SG-001": false, "TFGUARD-IAM-001": false,
	}
	for _, f := range r.Findings {
		if _, ok := want[f.ID]; ok {
			want[f.ID] = true
		}
	}
	for id, seen := range want {
		if !seen {
			t.Fatalf("missing %s; got %+v", id, r.Findings)
		}
	}
}

func TestScanOPAOnly(t *testing.T) {
	t.Parallel()
	plan := &tfplan.Plan{
		FormatVersion: "1.2",
		ResourceChanges: []tfplan.ResourceChange{{
			Address: "aws_ebs_volume.data",
			Mode:    "managed",
			Type:    "aws_ebs_volume",
			Change:  tfplan.Change{Actions: []string{"create"}, After: json.RawMessage(`{"encrypted":false}`)},
		}},
	}
	r, err := Scan(context.Background(), plan, ScanOptions{SkipCheckov: true, SkipTfsec: true})
	if err != nil || len(r.Findings) != 1 || r.Findings[0].ID != "TFGUARD-EBS-001" {
		t.Fatalf("got %+v err=%v", r, err)
	}
}
