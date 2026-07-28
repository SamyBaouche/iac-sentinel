# Require EBS volumes to be encrypted.
package tfguard.ebs_encryption

import rego.v1

violation contains finding if {
	some rc in input.resource_changes
	rc.type == "aws_ebs_volume"
	after := object.get(object.get(rc, "change", {}), "after", {})
	after != null
	not object.get(after, "encrypted", false)

	finding := {
		"id": "TFGUARD-EBS-001",
		"severity": "HIGH",
		"title": "EBS volume is not encrypted",
		"description": sprintf("%s should set encrypted = true", [rc.address]),
		"resource": rc.address,
	}
}
