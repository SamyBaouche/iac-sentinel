package policies

import "embed"

// FS embeds Rego policy files into the binary.
//
//go:embed *.rego
var FS embed.FS
