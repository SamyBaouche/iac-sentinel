# Deny IAM policy documents that allow Action = "*".
package sentinel.iam_wildcard

import rego.v1

violation contains finding if {
	some rc in input.resource_changes
	rc.type in {"aws_iam_policy", "aws_iam_role_policy", "aws_iam_user_policy"}
	after := object.get(object.get(rc, "change", {}), "after", {})
	doc := object.get(after, "policy", "")
	is_string(doc)
	has_wildcard_action(doc)

	finding := {
		"id": "SENTINEL-IAM-001",
		"severity": "CRITICAL",
		"title": "IAM policy allows Action wildcard *",
		"description": sprintf("%s grants Action=\"*\" which is overly permissive", [rc.address]),
		"resource": rc.address,
	}
}

has_wildcard_action(doc) if contains(doc, `"Action":"*"`)

has_wildcard_action(doc) if contains(doc, `"Action": "*"`)
