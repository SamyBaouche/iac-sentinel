// Package policy runs security checks and returns a unified Finding list.
// Sources: optional Checkov/tfsec CLIs, and embedded OPA Rego policies.
package policy

// Severity is how serious a finding is.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityUnknown  Severity = "UNKNOWN"
)

// Source identifies which scanner produced a finding.
type Source string

const (
	SourceCheckov Source = "checkov"
	SourceTfsec   Source = "tfsec"
	SourceOPA     Source = "opa"
)

// Finding is the shared shape for every scanner result.
type Finding struct {
	ID          string
	Source      Source
	Severity    Severity
	Title       string
	Description string
	Resource    string
	File        string
	Guideline   string
}

// Result is findings plus soft warnings (e.g. missing optional CLI).
type Result struct {
	Findings []Finding
	Warnings []string
}
