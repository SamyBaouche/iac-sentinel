// =============================================================================
// FILE: risk_test.go
// PURPOSE: Automatically check that risk.go behaves correctly.
//
// In Go, a file named "*_test.go" is a TEST file.
// Run all tests from the project root with:
//
//   go test ./internal/risk/
//
// Rules for test functions:
//   - name must start with "Test"
//   - take one argument: t *testing.T  (Go's test helper object)
//   - live in the same package (here: "risk") so they can call private funcs
//
// These tests use a common Go pattern called "table-driven tests":
//   1. put many cases in a list (input → expected output)
//   2. loop over the list
//   3. fail the test if actual result != expected
// =============================================================================

package risk

// Import multiple packages inside parentheses.
//   "testing" — Go's built-in test library
//   tfplan    — so we can write tfplan.ActionCreate, etc.
import (
	"testing"

	"github.com/SamyBaouche/iac-sentinel/internal/tfplan"
)

// TestLevelString checks that Level.String() returns the right text.
func TestLevelString(t *testing.T) {
	// t.Parallel() = this test may run at the same time as other tests (faster).
	t.Parallel()

	// "tests" is a SLICE (dynamic list) of anonymous STRUCTS.
	// A struct is a group of named fields — like a tiny object / row.
	// Each row = one test case: which Level in, which string out.
	tests := []struct {
		level Level  // input
		want  string // expected output
	}{
		{SAFE, "SAFE"},
		{CAUTION, "CAUTION"},
		{DANGER, "DANGER"},
		{CRITICAL, "CRITICAL"},
		{Level(99), "UNKNOWN"}, // weird value must not crash
	}

	// "for _, tt := range tests" means:
	//   loop over every item in tests
	//   "_"     = ignore the index (0, 1, 2, …)
	//   "tt"    = the current test case (short for "test table row")
	for _, tt := range tests {
		// t.Run creates a NAMED sub-test (nice error messages).
		// The second argument is an anonymous function (a function with no name).
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			// Call the code under test.
			got := tt.level.String()

			// If result is wrong, fail loudly with a clear message.
			// %q wraps strings in quotes in the error output.
			if got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestIsStateful checks which resource types count as "stateful".
func TestIsStateful(t *testing.T) {
	t.Parallel()

	tests := []struct {
		resourceType string
		want         bool // true = should be in the stateful table
	}{
		// These HOLD data → true
		{"aws_db_instance", true},
		{"aws_rds_cluster", true},
		{"aws_s3_bucket", true},
		{"aws_ebs_volume", true},
		{"aws_dynamodb_table", true},
		{"aws_efs_file_system", true},

		// These do NOT hold durable data in our table → false
		{"aws_instance", false},
		{"aws_security_group", false},
		{"aws_subnet", false},
		{"", false}, // empty string should be false
	}

	for _, tt := range tests {
		// Sub-test name cannot be empty — use "empty" as a fallback label.
		name := tt.resourceType
		if name == "" {
			name = "empty"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := IsStateful(tt.resourceType)
			if got != tt.want {
				// %v prints the value in a default format (works for bool, etc.)
				t.Fatalf("IsStateful(%q) = %v, want %v", tt.resourceType, got, tt.want)
			}
		})
	}
}

// TestClassify is the most important test: it checks the full risk rules.
func TestClassify(t *testing.T) {
	t.Parallel()

	// Each case has a name (for humans), inputs, and the Level we expect.
	tests := []struct {
		name         string        // label shown if the test fails
		action       tfplan.Action // what Terraform plans to do
		resourceType string        // which AWS resource
		want         Level         // expected risk
	}{
		// --- Non-stateful resources: use the BASE mapping only ---
		{name: "create instance", action: tfplan.ActionCreate, resourceType: "aws_instance", want: SAFE},
		{name: "no-op instance", action: tfplan.ActionNoOp, resourceType: "aws_instance", want: SAFE},
		{name: "read data", action: tfplan.ActionRead, resourceType: "aws_ami", want: SAFE},
		{name: "unknown action", action: "", resourceType: "aws_instance", want: SAFE},
		{name: "update instance", action: tfplan.ActionUpdate, resourceType: "aws_instance", want: CAUTION},
		{name: "replace security group", action: tfplan.ActionReplace, resourceType: "aws_security_group", want: DANGER},
		{name: "delete instance", action: tfplan.ActionDelete, resourceType: "aws_instance", want: DANGER},

		// --- Stateful resources: BASE level, then +1 ---
		// create: SAFE→CAUTION | update: CAUTION→DANGER | delete/replace: DANGER→CRITICAL
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
				// %s calls String() on Level, so you see "DANGER" not "2"
				t.Fatalf("Classify(%q, %q) = %s, want %s", tt.action, tt.resourceType, got, tt.want)
			}
		})
	}
}

// TestEscalateCapsAtCritical checks the private helper escalate().
// Tests in the same package CAN call private (lowercase) functions.
func TestEscalateCapsAtCritical(t *testing.T) {
	t.Parallel()

	// Already CRITICAL → stay CRITICAL (do not go to 4).
	if got := escalate(CRITICAL); got != CRITICAL {
		t.Fatalf("escalate(CRITICAL) = %s, want CRITICAL", got)
	}

	// DANGER + 1 → CRITICAL
	if got := escalate(DANGER); got != CRITICAL {
		t.Fatalf("escalate(DANGER) = %s, want CRITICAL", got)
	}
}
