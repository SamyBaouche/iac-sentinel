// Package risk classifies how dangerous a Terraform change is.
package risk

import "github.com/SamyBaouche/tfguard/internal/tfplan"

// Level is risk severity from safest to worst.
type Level int

const (
	SAFE Level = iota
	CAUTION
	DANGER
	CRITICAL
)

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

// statefulTypes are AWS resources that hold durable data.
// Destroying or replacing them can cause irreversible loss, so we escalate risk.
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

// IsStateful reports whether resourceType stores durable data.
func IsStateful(resourceType string) bool {
	_, ok := statefulTypes[resourceType]
	return ok
}

// Classify maps an action to a Level, then escalates +1 for stateful resources.
//
//	create / no-op / read → SAFE
//	update               → CAUTION
//	replace / delete     → DANGER
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

func escalate(l Level) Level {
	if l >= CRITICAL {
		return CRITICAL
	}
	return l + 1
}
