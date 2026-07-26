# IaC Sentinel

Automated reviewer for Terraform plans. IaC Sentinel reads `terraform plan -json` output and surfaces destructive changes, policy violations, estimated cost impact, and a plain-language summary so teams do not ship risky infrastructure changes unnoticed.

**Current status:** plan parsing (`internal/tfplan`) and risk classification (`internal/risk`) are implemented and tested. CLI rendering, `--fail-on`, policy checks, cost estimation, and CI integration are planned next.

## Problem

CI often stops at "terraform plan succeeded." Nobody reads a 400-line plan diff. A single `destroy` on a production database can slip through. IaC Sentinel is meant to catch that before merge.

## Scope (v1)

- AWS only
- Fully usable without any AI component (`--no-ai` when the explainer lands)
- Optional external scanners (Checkov, tfsec): warn if missing, never hard-fail solely because a binary is absent
- Idiomatic Go: wrapped errors, table-driven tests, `gofmt` / `go vet` clean

## Architecture

```
terraform plan -json
        |
        v
   [ PARSER ]     decode resource_changes into typed Go structs   ✓ done
        |
        v
   [ ANALYZER ]   risk ✓ | policies | cost | ML score             (partial)
        |
        v
   [ EXPLAINER ]  optional LLM summary                            (upcoming)
        |
        v
   [ RENDERER ]   terminal table / PR markdown                    (upcoming)
```

## Repository layout

```
iac-sentinel/
├── go.mod
├── Makefile
├── README.md
├── internal/
│   ├── tfplan/          # plan JSON parsing and summary
│   │   ├── types.go
│   │   ├── parse.go
│   │   ├── summary.go
│   │   └── tfplan_test.go
│   └── risk/            # SAFE → CRITICAL classification
│       ├── risk.go
│       └── risk_test.go
└── testdata/            # fixture plans for unit tests
```

## What works today

### `internal/tfplan`

| Piece | Responsibility |
|-------|----------------|
| `types.go` | Typed `Plan`, `ResourceChange`, `Change`; collapses Terraform action lists into a single `Action` (including replace as `["delete","create"]` or `["create","delete"]`) |
| `parse.go` | `Parse` / `ParseFile`; returns `ErrNotAPlan` when `format_version` is missing |
| `summary.go` | Counts create/update/replace/delete; excludes no-ops and data sources; sorts changes by address for stable output |

`Before` and `After` stay as `json.RawMessage` so later stages can decode only the fields they need (cost attributes, policy input, and so on).

### `internal/risk`

| Piece | Responsibility |
|-------|----------------|
| `Level` (`iota`) | `SAFE` → `CAUTION` → `DANGER` → `CRITICAL`, plus `String()` for display |
| Stateful table | AWS types that hold durable data (RDS, S3, EBS, DynamoDB, EFS, ElastiCache, …) |
| `Classify(action, resourceType)` | Maps action → base level, then escalates +1 when the resource is stateful (capped at `CRITICAL`) |

**Base mapping (non-stateful):**

| Action | Level |
|--------|-------|
| create / no-op / read / unknown | `SAFE` |
| update | `CAUTION` |
| replace / delete | `DANGER` |

**Examples with escalation:** create S3 → `CAUTION`; update RDS → `DANGER`; delete RDS → `CRITICAL`.

## Getting started

### Prerequisites

- Go 1.26+

### Build and test

```bash
make test
# or
go test ./... -cover

go vet ./...
gofmt -l .
go build ./...
```

There is no CLI entrypoint yet. Exercise the packages via unit tests and the fixtures under `testdata/`.

### Example: load a fixture in tests

Fixtures such as `testdata/plan_mixed.json` cover create, update, replace, delete, data sources, and no-ops. `Summarize` should report one of each mutating action and omit reads/no-ops.

### Example: classify a change

```go
import (
    "github.com/SamyBaouche/iac-sentinel/internal/risk"
    "github.com/SamyBaouche/iac-sentinel/internal/tfplan"
)

level := risk.Classify(tfplan.ActionDelete, "aws_db_instance")
// level == risk.CRITICAL
fmt.Println(level.String()) // "CRITICAL"
```

## Roadmap

1. Terminal renderer + `scan` / `version` CLI
2. Wire risk into the CLI with `--fail-on` (classification package is done)
3. Policy engine (OPA Rego + optional Checkov/tfsec)
4. Static AWS cost delta estimation
5. Embedded logistic-regression risk score
6. Optional LLM explainer (Ollama / HTTP), disabled with `--no-ai`
7. GitHub Action (PR comment) and GoReleaser distribution

## Development notes

- Prefer table-driven tests and fixtures over live Terraform in unit tests
- Wrap errors with `%w` and keep sentinel errors (`errors.Is`) for control flow
- Keep packages under `internal/` until a public API is intentionally exported
- `internal/risk` includes beginner-oriented comments explaining Go syntax alongside the logic

## License

License file to be added (Apache 2.0 intended).
