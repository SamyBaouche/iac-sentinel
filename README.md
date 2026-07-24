# IaC Sentinel

Automated reviewer for Terraform plans. IaC Sentinel reads `terraform plan -json` output and surfaces destructive changes, policy violations, estimated cost impact, and a plain-language summary so teams do not ship risky infrastructure changes unnoticed.

**Current status:** the Terraform plan parser (`internal/tfplan`) is implemented and tested. CLI rendering, risk classification, policy checks, cost estimation, and CI integration are planned next.

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
   [ PARSER ]     decode resource_changes into typed Go structs
        |
        v
   [ ANALYZER ]   risk | policies | cost | ML score  (upcoming)
        |
        v
   [ EXPLAINER ]  optional LLM summary               (upcoming)
        |
        v
   [ RENDERER ]   terminal table / PR markdown       (upcoming)
```

## Repository layout

```
iac-sentinel/
├── go.mod
├── Makefile
├── README.md
├── internal/
│   └── tfplan/          # plan JSON parsing and summary
│       ├── types.go
│       ├── parse.go
│       ├── summary.go
│       └── tfplan_test.go
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

There is no CLI entrypoint yet. Exercise the parser via unit tests and the fixtures under `testdata/`.

### Example: load a fixture in tests

Fixtures such as `testdata/plan_mixed.json` cover create, update, replace, delete, data sources, and no-ops. `Summarize` should report one of each mutating action and omit reads/no-ops.

## Roadmap

1. Terminal renderer + `scan` / `version` CLI
2. Risk classification (`SAFE` → `CRITICAL`) with `--fail-on`
3. Policy engine (OPA Rego + optional Checkov/tfsec)
4. Static AWS cost delta estimation
5. Embedded logistic-regression risk score
6. Optional LLM explainer (Ollama / HTTP), disabled with `--no-ai`
7. GitHub Action (PR comment) and GoReleaser distribution

## Development notes

- Prefer table-driven tests and fixtures over live Terraform in unit tests
- Wrap errors with `%w` and keep sentinel errors (`errors.Is`) for control flow
- Keep packages under `internal/` until a public API is intentionally exported

## License

License file to be added (Apache 2.0 intended).
