// Package risk classifies the severity of planned Terraform resource changes.
package risk

import "github.com/SamyBaouche/iac-sentinel/internal/tfplan"

// Level is the risk severity of a planned change.
type Level int

const (
	SAFE Level = iota
	CAUTION
	DANGER
	CRITICAL
)

// String returns the display name of the risk level.
func (l Level) String() string {
	switch l {
	case SAFE:
		return "SAFE"
	case CAUTION:
		return "CAUTION"
	case DANGER:
		return "DANGER"
	case CRITICAL:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// statefulTypes lists AWS resource types that hold durable data.
// Destroying or replacing them can cause irreversible data loss.
var statefulTypes = map[string]struct{}{
	"aws_db_instance":                   {},
	"aws_rds_cluster":                   {},
	"aws_rds_cluster_instance":          {},
	"aws_s3_bucket":                     {},
	"aws_ebs_volume":                    {},
	"aws_ebs_snapshot":                  {},
	"aws_dynamodb_table":                {},
	"aws_efs_file_system":               {},
	"aws_elasticache_cluster":           {},
	"aws_elasticache_replication_group": {},
	"aws_redshift_cluster":              {},
	"aws_opensearch_domain":             {},
	"aws_elasticsearch_domain":          {},
	"aws_docdb_cluster":                 {},
	"aws_neptune_cluster":               {},
}

// IsStateful reports whether resourceType is a known stateful AWS resource.
func IsStateful(resourceType string) bool {
	_, ok := statefulTypes[resourceType]
	return ok
}

// Classify maps a Terraform action to a risk Level.
// Stateful resources (RDS, S3, EBS, DynamoDB, …) escalate one level,
// capped at CRITICAL.
//
// Base mapping:
//
//	create / no-op / read / unknown → SAFE
//	update                          → CAUTION
//	replace                         → DANGER
//	delete                          → DANGER
func Classify(action tfplan.Action, resourceType string) Level {
	level := baseLevel(action)
	if IsStateful(resourceType) {
		level = escalate(level)
	}
	return level
}

func baseLevel(action tfplan.Action) Level {
	switch action {
	case tfplan.ActionUpdate:
		return CAUTION
	case tfplan.ActionReplace, tfplan.ActionDelete:
		return DANGER
	default:
		// create, no-op, read, empty/unknown
		return SAFE
	}
}

func escalate(l Level) Level {
	if l >= CRITICAL {
		return CRITICAL
	}
	return l + 1
}
