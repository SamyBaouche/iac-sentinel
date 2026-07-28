// Package risk classifies how dangerous a planned Terraform change is.
//
// Used by the CLI before --fail-on decides whether the process should exit 1.
package risk

import "github.com/SamyBaouche/tfguard/internal/tfplan"

// Level is risk severity. Values are ordered so escalate can use level+1.
type Level int

const (
	SAFE     Level = iota // create / read / no-op
	CAUTION               // update
	DANGER                // replace or delete
	CRITICAL              // worst case (e.g. delete a stateful resource)
)

// String returns the display name used in the terminal report.
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
// Replacing or deleting them can cause irreversible data loss.
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

// IsStateful reports whether resourceType is in the stateful table.
func IsStateful(resourceType string) bool {
	_, ok := statefulTypes[resourceType]
	return ok
}

// Classify maps action → base level, then escalates +1 when the resource is stateful.
//
//	create / no-op / read → SAFE
//	update               → CAUTION
//	replace / delete     → DANGER
//
// Example: delete aws_db_instance → DANGER then escalate → CRITICAL.
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
		return SAFE
	}
}

// escalate raises severity by one step, never past CRITICAL.
func escalate(l Level) Level {
	if l >= CRITICAL {
		return CRITICAL
	}
	return l + 1
}
