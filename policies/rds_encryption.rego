# Require RDS instances to enable storage encryption.
package tfguard.rds_encryption

import rego.v1

violation contains finding if {
	some rc in input.resource_changes
	rc.type == "aws_db_instance"
	after := object.get(object.get(rc, "change", {}), "after", {})
	after != null
	not object.get(after, "storage_encrypted", false)

	finding := {
		"id": "TFGUARD-RDS-001",
		"severity": "HIGH",
		"title": "RDS instance storage is not encrypted",
		"description": sprintf("%s should set storage_encrypted = true", [rc.address]),
		"resource": rc.address,
	}
}
