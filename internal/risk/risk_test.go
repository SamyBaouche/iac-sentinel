package risk

import (
	"testing"

	"github.com/SamyBaouche/tfguard/internal/tfplan"
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
		if got := tt.level.String(); got != tt.want {
			t.Fatalf("String() = %q, want %q", got, tt.want)
		}
	}
}

func TestIsStateful(t *testing.T) {
	t.Parallel()
	if !IsStateful("aws_db_instance") || IsStateful("aws_instance") {
		t.Fatal("unexpected IsStateful results")
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
		{"create instance", tfplan.ActionCreate, "aws_instance", SAFE},
		{"update instance", tfplan.ActionUpdate, "aws_instance", CAUTION},
		{"delete instance", tfplan.ActionDelete, "aws_instance", DANGER},
		{"replace sg", tfplan.ActionReplace, "aws_security_group", DANGER},
		{"create s3", tfplan.ActionCreate, "aws_s3_bucket", CAUTION},
		{"update rds", tfplan.ActionUpdate, "aws_db_instance", DANGER},
		{"delete rds", tfplan.ActionDelete, "aws_db_instance", CRITICAL},
		{"replace dynamodb", tfplan.ActionReplace, "aws_dynamodb_table", CRITICAL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Classify(tt.action, tt.resourceType); got != tt.want {
				t.Fatalf("Classify(%q, %q) = %s, want %s", tt.action, tt.resourceType, got, tt.want)
			}
		})
	}
}

func TestEscalateCapsAtCritical(t *testing.T) {
	t.Parallel()
	if escalate(CRITICAL) != CRITICAL || escalate(DANGER) != CRITICAL {
		t.Fatal("escalate must cap at CRITICAL")
	}
}
