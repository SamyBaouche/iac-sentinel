package tfplan

import "encoding/json"

// Action is a collapsed representation of a Terraform change action list.
type Action string

const (
	ActionNoOp    Action = "no-op"
	ActionCreate  Action = "create"
	ActionRead    Action = "read"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionReplace Action = "replace"
)

// Plan is the subset of a terraform plan JSON document needed for review.
type Plan struct {
	FormatVersion    string           `json:"format_version"`
	TerraformVersion string           `json:"terraform_version"`
	ResourceChanges  []ResourceChange `json:"resource_changes"`
}

// ResourceChange describes a single planned change to a managed resource or data source.
type ResourceChange struct {
	Address      string `json:"address"`
	Mode         string `json:"mode"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	ProviderName string `json:"provider_name"`
	Change       Change `json:"change"`
}

// Change holds the before/after values and the raw Terraform actions list.
// Before and After remain json.RawMessage to avoid provider-specific schemas.
type Change struct {
	Actions []string        `json:"actions"`
	Before  json.RawMessage `json:"before"`
	After   json.RawMessage `json:"after"`
}

// Action collapses Terraform's actions array into a single Action value.
// A replace is encoded as ["delete","create"] or ["create","delete"]
// (create_before_destroy); both return ActionReplace.
// Empty or unrecognized action lists return the empty Action "".
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
