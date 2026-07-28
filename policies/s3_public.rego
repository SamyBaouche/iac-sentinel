# Deny S3 buckets configured with a public ACL.
package tfguard.s3_public

import rego.v1

violation contains finding if {
	some rc in input.resource_changes
	rc.type == "aws_s3_bucket"
	after := object.get(rc, "change", {})
	after_vals := object.get(after, "after", {})
	acl := object.get(after_vals, "acl", "")
	acl in {"public-read", "public-read-write", "authenticated-read"}

	finding := {
		"id": "TFGUARD-S3-001",
		"severity": "HIGH",
		"title": "S3 bucket ACL allows public access",
		"description": sprintf("Bucket %s uses ACL %q which can expose objects publicly", [rc.address, acl]),
		"resource": rc.address,
	}
}
