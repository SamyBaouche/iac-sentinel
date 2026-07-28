// Package tfplan parses terraform plan JSON into typed Go values.
package tfplan

import "encoding/json"

// Action is one Terraform change, collapsed from the raw actions array.
type Action string

const (
	ActionNoOp    Action = "no-op"
	ActionCreate  Action = "create"
	ActionRead    Action = "read"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionReplace Action = "replace"
)

// Plan is the subset of a terraform plan JSON document we need for review.
type Plan struct {
	FormatVersion    string           `json:"format_version"`
	TerraformVersion string           `json:"terraform_version"`
	ResourceChanges  []ResourceChange `json:"resource_changes"`
}

// ResourceChange is one planned change to a managed resource or data source.
type ResourceChange struct {
	Address      string `json:"address"` // e.g. aws_db_instance.main
	Mode         string `json:"mode"`    // "managed" or "data"
	Type         string `json:"type"`    // e.g. aws_s3_bucket
	Name         string `json:"name"`
	ProviderName string `json:"provider_name"`
	Change       Change `json:"change"`
}

// Change holds before/after values.
// Before/After stay as RawMessage so later stages decode only what they need.
type Change struct {
	Actions []string        `json:"actions"`
	Before  json.RawMessage `json:"before"`
	After   json.RawMessage `json:"after"`
}

// Action collapses Terraform's actions list into a single value.
// A replace is encoded as ["delete","create"] or ["create","delete"].
// Empty or unrecognized lists return "".
func (c Change) Action() Action {
	switch len(c.Actions) {
	case 1:
		switch c.Actions[0] {
		case "no-op":
			return ActionNoOp
		case "create":
			return ActionCreate
		case "read":
			return ActionRead
		case "update":
			return ActionUpdate
		case "delete":
			return ActionDelete
		default:
			return ""
		}
	case 2:
		a, b := c.Actions[0], c.Actions[1]
		if (a == "delete" && b == "create") || (a == "create" && b == "delete") {
			return ActionReplace
		}
		return ""
	default:
		return ""
	}
}
