// Package policy runs security checks on Terraform code / plans and returns
// a single shared format: Finding.
//
// Three sources feed into Finding:
//  1. Checkov  — external CLI scanner (optional)
//  2. tfsec    — external CLI scanner (optional)
//  3. OPA      — policies written in Rego, evaluated in-process
//
// If Checkov or tfsec is not installed, we WARN and continue.
// We never crash only because an optional scanner is missing.
package policy

// Severity is how serious a finding is (shared across all scanners).
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityUnknown  Severity = "UNKNOWN"
)

// Source says which tool produced the finding.
type Source string

const (
	SourceCheckov Source = "checkov"
	SourceTfsec   Source = "tfsec"
	SourceOPA     Source = "opa"
)

// Finding is the UNIFIED shape for every security issue we report.
// Checkov, tfsec, and OPA all get converted into this struct so the rest
// of the app (CLI, PR comments, --fail-on) only speaks one language.
type Finding struct {
	ID          string   // e.g. "CKV_AWS_20", "AVD-AWS-0086", "SENTINEL-S3-001"
	Source      Source   // checkov | tfsec | opa
	Severity    Severity // CRITICAL / HIGH / MEDIUM / LOW
	Title       string   // short human label
	Description string   // longer explanation
	Resource    string   // e.g. "aws_s3_bucket.logs"
	File        string   // path in the Terraform tree, if known
	Guideline   string   // link or hint on how to fix
}

// Result is what a scanner returns: findings + soft warnings.
type Result struct {
	Findings []Finding
	Warnings []string // e.g. "checkov not found on PATH; skipping"
}
