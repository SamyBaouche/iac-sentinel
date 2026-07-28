// Package policy runs security checks and normalizes them into Finding values.
//
// Three sources feed the same shape:
//   - Checkov (optional external CLI)
//   - tfsec   (optional external CLI)
//   - OPA     (embedded Rego under policies/)
//
// Missing optional CLIs produce Warnings only — they do not fail the scan.
package policy

// Severity is how serious a finding is (scanner-native scale).
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityUnknown  Severity = "UNKNOWN"
)

// Source identifies which tool produced the finding.
type Source string

const (
	SourceCheckov Source = "checkov"
	SourceTfsec   Source = "tfsec"
	SourceOPA     Source = "opa"
)

// Finding is the unified issue shape used by the CLI and --fail-on mapping.
type Finding struct {
	ID          string // e.g. CKV_AWS_20, TFGUARD-S3-001
	Source      Source
	Severity    Severity
	Title       string
	Description string
	Resource    string // Terraform address when known
	File        string
	Guideline   string
}

// Result holds findings plus soft warnings (missing binary, etc.).
type Result struct {
	Findings []Finding
	Warnings []string
}
