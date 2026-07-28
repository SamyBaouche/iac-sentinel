package policies

import "embed"

// FS embeds every .rego file in this folder into the Go binary.
// That way OPA policies ship with the app — no need to copy files at runtime.
//
//go:embed *.rego
var FS embed.FS
