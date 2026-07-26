// =============================================================================
// FILE: risk.go
// PURPOSE: Decide how dangerous a Terraform infrastructure change is.
//
// You do NOT need to know Go deeply to follow this file. Read the comments
// top-to-bottom — they explain both the *idea* and the Go *syntax*.
//
// Big picture in this project:
//   1. tfplan  → reads Terraform's plan JSON ("what will change?")
//   2. risk    → THIS FILE ("how risky is each change?")
//   3. later   → fail CI if risk is too high
//
// Example:
//   delete an EC2 server     → DANGER
//   delete an RDS database   → CRITICAL  (data can be lost forever)
// =============================================================================

// Every Go file starts with "package <name>".
// Files in the same folder share the same package name and can call each
// other's functions without importing.
package risk

// "import" brings in code from another package.
// Here we need tfplan.Action (create, update, delete, …) defined elsewhere.
import "github.com/SamyBaouche/iac-sentinel/internal/tfplan"

// -----------------------------------------------------------------------------
// TYPES
// In Go, "type Name UnderlyingType" creates a new named type.
// Level is based on int, but is NOT the same as a plain int — the compiler
// will complain if you mix them by accident. That is a good thing.
// -----------------------------------------------------------------------------

// Level = how severe a change is (SAFE, CAUTION, DANGER, CRITICAL).
type Level int

// -----------------------------------------------------------------------------
// CONSTANTS + iota
// "const (...)" defines fixed values that never change.
//
// "iota" is a Go counter that starts at 0 and increases by 1 for each line:
//   SAFE     = 0
//   CAUTION  = 1
//   DANGER   = 2
//   CRITICAL = 3
//
// Why numbers? So we can raise risk with simple math: level + 1
// -----------------------------------------------------------------------------

const (
	SAFE     Level = iota // 0 — low risk (creating something new)
	CAUTION               // 1 — review carefully (updating config)
	DANGER                // 2 — destructive (replace or delete)
	CRITICAL              // 3 — worst case (e.g. delete a database)
)

// -----------------------------------------------------------------------------
// METHODS
// A method looks like a function, but has a "receiver" before the name:
//
//   func (l Level) String() string { ... }
//        ^^^^^^^
//        receiver = the value you call the method ON
//
// Call it like:  DANGER.String()  →  "DANGER"
//
// "string" after the () is the RETURN type (this function gives back text).
// -----------------------------------------------------------------------------

// String converts a Level number into readable text for the terminal / logs.
func (l Level) String() string {
	// "switch" is like many if/else checks on one value.
	switch l {
	case SAFE:
		return "SAFE" // "return" exits the function and gives back this value
	case CAUTION:
		return "CAUTION"
	case DANGER:
		return "DANGER"
	case CRITICAL:
		return "CRITICAL"
	default:
		// "default" runs if nothing matched (e.g. Level(99) from a bug).
		return "UNKNOWN"
	}
}

// -----------------------------------------------------------------------------
// VARIABLES + MAPS
// "var name = value" declares a package-level variable (usable by all funcs).
//
// A map is a dictionary / hash table:
//   map[KEY_TYPE]VALUE_TYPE
//
// We use map[string]struct{} as a SET of names:
//   - key   = resource type name ("aws_s3_bucket")
//   - value = empty struct {}  (we only care if the key EXISTS)
//
// Looking up a key:
//   value, ok := myMap["key"]
//   ok == true  → key was found
//   ok == false → key was missing
// -----------------------------------------------------------------------------

// statefulTypes lists AWS resources that STORE durable data.
// If Terraform deletes/replaces these, you can lose data → higher risk.
var statefulTypes = map[string]struct{}{
	// --- Databases ---
	"aws_db_instance":          {}, // RDS (one database server)
	"aws_rds_cluster":          {}, // Aurora / RDS cluster
	"aws_rds_cluster_instance": {},
	"aws_docdb_cluster":        {}, // DocumentDB
	"aws_neptune_cluster":      {}, // Graph DB
	"aws_redshift_cluster":     {}, // Data warehouse

	// --- Storage ---
	"aws_s3_bucket":       {}, // Files / objects
	"aws_ebs_volume":      {}, // Disk for EC2
	"aws_ebs_snapshot":    {}, // Disk backup
	"aws_efs_file_system": {}, // Shared filesystem

	// --- NoSQL / cache / search ---
	"aws_dynamodb_table":                {},
	"aws_elasticache_cluster":           {},
	"aws_elasticache_replication_group": {},
	"aws_opensearch_domain":             {},
	"aws_elasticsearch_domain":          {}, // older Terraform name
}

// -----------------------------------------------------------------------------
// FUNCTIONS
// Syntax:
//   func Name(paramName paramType) returnType { ... }
//
// Names that start with a CAPITAL letter are PUBLIC (other packages can call).
// Names that start with lowercase are PRIVATE (only this package).
//   IsStateful  → public
//   baseLevel   → private
// -----------------------------------------------------------------------------

// IsStateful asks: "Does this resource type hold durable data?"
//
// Input:  resourceType — e.g. "aws_db_instance"
// Output: true or false (Go's bool type)
func IsStateful(resourceType string) bool {
	// "_" means "I don't care about this value, throw it away".
	// We only need "ok" (was the key found?).
	_, ok := statefulTypes[resourceType]
	return ok
}

// Classify is the MAIN function of this file.
//
// Inputs:
//
//	action       — what Terraform will do (create, update, delete, …)
//	resourceType — which AWS resource (aws_instance, aws_s3_bucket, …)
//
// Output:
//
//	a Level (SAFE / CAUTION / DANGER / CRITICAL)
//
// Logic in two steps:
//  1. base level from the action alone
//  2. if stateful → raise by +1 (never above CRITICAL)
//
// Cheat sheet (non-stateful):
//
//	create / no-op / read  → SAFE
//	update                 → CAUTION
//	replace / delete       → DANGER
//
// Cheat sheet (stateful, after +1):
//
//	create S3     → CAUTION
//	update RDS    → DANGER
//	delete RDS    → CRITICAL
func Classify(action tfplan.Action, resourceType string) Level {
	// ":=" declares a new variable AND assigns it (short form of var).
	level := baseLevel(action)

	// "if" runs the block only when the condition is true.
	if IsStateful(resourceType) {
		level = escalate(level) // overwrite level with the higher one
	}
	return level
}

// baseLevel looks ONLY at the action (not whether data is stored).
// Private helper used by Classify.
func baseLevel(action tfplan.Action) Level {
	switch action {
	case tfplan.ActionUpdate:
		return CAUTION
	case tfplan.ActionReplace, tfplan.ActionDelete:
		// Two cases share the same result (comma-separated).
		// replace = destroy old + create new
		// delete  = remove completely
		return DANGER
	default:
		// create, no-op, read, or anything unknown → SAFE
		return SAFE
	}
}

// escalate moves risk one step up, but stops at CRITICAL.
// Because SAFE=0 … CRITICAL=3, "l + 1" means "next worse level".
func escalate(l Level) Level {
	// ">=" means "greater than or equal".
	if l >= CRITICAL {
		return CRITICAL // already at the top — stay there
	}
	return l + 1
}
