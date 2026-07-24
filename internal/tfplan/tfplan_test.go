package tfplan

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testdata(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestChangeAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		actions []string
		want    Action
	}{
		{name: "no-op", actions: []string{"no-op"}, want: ActionNoOp},
		{name: "create", actions: []string{"create"}, want: ActionCreate},
		{name: "read", actions: []string{"read"}, want: ActionRead},
		{name: "update", actions: []string{"update"}, want: ActionUpdate},
		{name: "delete", actions: []string{"delete"}, want: ActionDelete},
		{name: "replace delete-create", actions: []string{"delete", "create"}, want: ActionReplace},
		{name: "replace create-delete", actions: []string{"create", "delete"}, want: ActionReplace},
		{name: "empty", actions: nil, want: ""},
		{name: "unknown single", actions: []string{"something"}, want: ""},
		{name: "unknown pair", actions: []string{"update", "create"}, want: ""},
		{name: "three actions", actions: []string{"delete", "create", "update"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Change{Actions: tt.actions}.Action()
			if got != tt.want {
				t.Fatalf("Action() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseMinimal(t *testing.T) {
	t.Parallel()

	plan, err := ParseFile(testdata("plan_minimal.json"))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if plan.FormatVersion != "1.2" {
		t.Fatalf("FormatVersion = %q, want 1.2", plan.FormatVersion)
	}
	if plan.TerraformVersion != "1.9.0" {
		t.Fatalf("TerraformVersion = %q, want 1.9.0", plan.TerraformVersion)
	}
	if len(plan.ResourceChanges) != 1 {
		t.Fatalf("len(ResourceChanges) = %d, want 1", len(plan.ResourceChanges))
	}
	rc := plan.ResourceChanges[0]
	if rc.Address != "aws_s3_bucket.logs" {
		t.Fatalf("Address = %q, want aws_s3_bucket.logs", rc.Address)
	}
	if rc.Change.Action() != ActionCreate {
		t.Fatalf("Action() = %q, want %q", rc.Change.Action(), ActionCreate)
	}
	if string(rc.Change.After) == "" || string(rc.Change.After) == "null" {
		t.Fatalf("After should be populated, got %s", rc.Change.After)
	}
}

func TestParseReplaceBothOrders(t *testing.T) {
	t.Parallel()

	plan, err := ParseFile(testdata("plan_replace.json"))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(plan.ResourceChanges) != 2 {
		t.Fatalf("len(ResourceChanges) = %d, want 2", len(plan.ResourceChanges))
	}
	for _, rc := range plan.ResourceChanges {
		if got := rc.Change.Action(); got != ActionReplace {
			t.Fatalf("%s Action() = %q, want %q", rc.Address, got, ActionReplace)
		}
	}
}

func TestParseNotAPlan(t *testing.T) {
	t.Parallel()

	_, err := ParseFile(testdata("plan_not_a_plan.json"))
	if !errors.Is(err, ErrNotAPlan) {
		t.Fatalf("error = %v, want ErrNotAPlan", err)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := Parse(strings.NewReader(`{not-json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "tfplan: decode") {
		t.Fatalf("error = %q, want wrapped decode error", err)
	}
}

func TestParseMissingFormatVersionFromReader(t *testing.T) {
	t.Parallel()

	_, err := Parse(strings.NewReader(`{"terraform_version":"1.0.0"}`))
	if !errors.Is(err, ErrNotAPlan) {
		t.Fatalf("error = %v, want ErrNotAPlan", err)
	}
}

func TestParseFileNotFound(t *testing.T) {
	t.Parallel()

	_, err := ParseFile(testdata("does_not_exist.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v, want os.ErrNotExist", err)
	}
}

func TestSummarizeMixed(t *testing.T) {
	t.Parallel()

	plan, err := ParseFile(testdata("plan_mixed.json"))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	s := Summarize(plan)
	if s.Creates != 1 || s.Updates != 1 || s.Replaces != 1 || s.Deletes != 1 {
		t.Fatalf("counts create=%d update=%d replace=%d delete=%d, want 1/1/1/1",
			s.Creates, s.Updates, s.Replaces, s.Deletes)
	}
	if len(s.Changes) != 4 {
		t.Fatalf("len(Changes) = %d, want 4", len(s.Changes))
	}

	wantOrder := []string{
		"aws_db_instance.main",
		"aws_instance.web",
		"aws_s3_bucket.logs",
		"aws_security_group.web",
	}
	for i, addr := range wantOrder {
		if s.Changes[i].Address != addr {
			t.Fatalf("Changes[%d].Address = %q, want %q", i, s.Changes[i].Address, addr)
		}
	}
}

func TestSummarizeExcludesNoOpAndData(t *testing.T) {
	t.Parallel()

	plan := &Plan{
		FormatVersion: "1.2",
		ResourceChanges: []ResourceChange{
			{
				Address: "aws_subnet.private",
				Mode:    "managed",
				Change:  Change{Actions: []string{"no-op"}},
			},
			{
				Address: "data.aws_ami.ubuntu",
				Mode:    "data",
				Change:  Change{Actions: []string{"read"}},
			},
			{
				Address: "aws_instance.web",
				Mode:    "managed",
				Change:  Change{Actions: []string{"create"}},
			},
		},
	}

	s := Summarize(plan)
	if s.Creates != 1 || s.Updates != 0 || s.Replaces != 0 || s.Deletes != 0 {
		t.Fatalf("unexpected counts: %+v", s)
	}
	if len(s.Changes) != 1 || s.Changes[0].Address != "aws_instance.web" {
		t.Fatalf("Changes = %+v, want only aws_instance.web", s.Changes)
	}
}

func TestSummarizeNilPlan(t *testing.T) {
	t.Parallel()

	s := Summarize(nil)
	if s.Creates != 0 || s.Updates != 0 || s.Replaces != 0 || s.Deletes != 0 || len(s.Changes) != 0 {
		t.Fatalf("nil plan should yield zero Summary, got %+v", s)
	}
}

func TestSummarizeEmptyPlan(t *testing.T) {
	t.Parallel()

	s := Summarize(&Plan{FormatVersion: "1.2"})
	if s.Creates != 0 || len(s.Changes) != 0 {
		t.Fatalf("empty plan should yield zero Summary, got %+v", s)
	}
}

func TestSummarizeStableSort(t *testing.T) {
	t.Parallel()

	plan := &Plan{
		FormatVersion: "1.2",
		ResourceChanges: []ResourceChange{
			{Address: "z_resource", Mode: "managed", Change: Change{Actions: []string{"create"}}},
			{Address: "a_resource", Mode: "managed", Change: Change{Actions: []string{"delete"}}},
			{Address: "m_resource", Mode: "managed", Change: Change{Actions: []string{"update"}}},
		},
	}

	s := Summarize(plan)
	want := []string{"a_resource", "m_resource", "z_resource"}
	for i, addr := range want {
		if s.Changes[i].Address != addr {
			t.Fatalf("Changes[%d].Address = %q, want %q", i, s.Changes[i].Address, addr)
		}
	}
}
