# Deny security group rules that open ingress to the entire Internet.
package tfguard.sg_open

import rego.v1

violation contains finding if {
	some rc in input.resource_changes
	rc.type in {"aws_security_group", "aws_security_group_rule"}
	after := object.get(object.get(rc, "change", {}), "after", {})

	# aws_security_group: look at ingress blocks
	some ingress in object.get(after, "ingress", [])
	"0.0.0.0/0" in object.get(ingress, "cidr_blocks", [])

	finding := {
		"id": "TFGUARD-SG-001",
		"severity": "CRITICAL",
		"title": "Security group open to 0.0.0.0/0",
		"description": sprintf("%s allows inbound traffic from the entire Internet", [rc.address]),
		"resource": rc.address,
	}
}

violation contains finding if {
	some rc in input.resource_changes
	rc.type == "aws_security_group_rule"
	after := object.get(object.get(rc, "change", {}), "after", {})
	object.get(after, "type", "") == "ingress"
	"0.0.0.0/0" in object.get(after, "cidr_blocks", [])

	finding := {
		"id": "TFGUARD-SG-001",
		"severity": "CRITICAL",
		"title": "Security group rule open to 0.0.0.0/0",
		"description": sprintf("%s allows inbound traffic from the entire Internet", [rc.address]),
		"resource": rc.address,
	}
}
