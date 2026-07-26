package risk

import (
	"testing"

	"github.com/SamyBaouche/iac-sentinel/internal/tfplan"
)

func TestLevelString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level Level
		want  string
	}{
		{SAFE, "SAFE"},
		{CAUTION, "CAUTION"},
		{DANGER, "DANGER"},
		{CRITICAL, "CRITICAL"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.level.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsStateful(t *testing.T) {
	t.Parallel()

	tests := []struct {
		resourceType string
		want         bool
	}{
		{"aws_db_instance", true},
		{"aws_rds_cluster", true},
		{"aws_s3_bucket", true},
		{"aws_ebs_volume", true},
		{"aws_dynamodb_table", true},
		{"aws_efs_file_system", true},
		{"aws_instance", false},
		{"aws_security_group", false},
		{"aws_subnet", false},
		{"", false},
	}

	for _, tt := range tests {
		name := tt.resourceType
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := IsStateful(tt.resourceType); got != tt.want {
				t.Fatalf("IsStateful(%q) = %v, want %v", tt.resourceType, got, tt.want)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		action       tfplan.Action
		resourceType string
		want         Level
	}{
		// Base levels — non-stateful
		{name: "create instance", action: tfplan.ActionCreate, resourceType: "aws_instance", want: SAFE},
		{name: "no-op instance", action: tfplan.ActionNoOp, resourceType: "aws_instance", want: SAFE},
		{name: "read data", action: tfplan.ActionRead, resourceType: "aws_ami", want: SAFE},
		{name: "unknown action", action: "", resourceType: "aws_instance", want: SAFE},
		{name: "update instance", action: tfplan.ActionUpdate, resourceType: "aws_instance", want: CAUTION},
		{name: "replace security group", action: tfplan.ActionReplace, resourceType: "aws_security_group", want: DANGER},
		{name: "delete instance", action: tfplan.ActionDelete, resourceType: "aws_instance", want: DANGER},

		// Escalation — stateful
		{name: "create s3", action: tfplan.ActionCreate, resourceType: "aws_s3_bucket", want: CAUTION},
		{name: "update rds", action: tfplan.ActionUpdate, resourceType: "aws_db_instance", want: DANGER},
		{name: "replace dynamodb", action: tfplan.ActionReplace, resourceType: "aws_dynamodb_table", want: CRITICAL},
		{name: "delete rds", action: tfplan.ActionDelete, resourceType: "aws_db_instance", want: CRITICAL},
		{name: "delete ebs", action: tfplan.ActionDelete, resourceType: "aws_ebs_volume", want: CRITICAL},
		{name: "replace s3", action: tfplan.ActionReplace, resourceType: "aws_s3_bucket", want: CRITICAL},
		{name: "update efs", action: tfplan.ActionUpdate, resourceType: "aws_efs_file_system", want: DANGER},
		{name: "create dynamodb", action: tfplan.ActionCreate, resourceType: "aws_dynamodb_table", want: CAUTION},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Classify(tt.action, tt.resourceType)
			if got != tt.want {
				t.Fatalf("Classify(%q, %q) = %s, want %s", tt.action, tt.resourceType, got, tt.want)
			}
		})
	}
}

func TestEscalateCapsAtCritical(t *testing.T) {
	t.Parallel()

	if got := escalate(CRITICAL); got != CRITICAL {
		t.Fatalf("escalate(CRITICAL) = %s, want CRITICAL", got)
	}
	if got := escalate(DANGER); got != CRITICAL {
		t.Fatalf("escalate(DANGER) = %s, want CRITICAL", got)
	}
}
