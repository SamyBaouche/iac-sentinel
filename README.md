# IaC Sentinel

Automated reviewer for Terraform plans. IaC Sentinel reads `terraform plan -json` output and surfaces destructive changes, policy violations, estimated cost impact, and a plain-language summary so teams do not ship risky infrastructure changes unnoticed.

**Current status:** plan parsing (`internal/tfplan`), risk classification (`internal/risk`), and the policy engine (`internal/policy` + OPA Rego, optional Checkov/tfsec) are implemented and tested. CLI rendering, `--fail-on`, cost estimation, and CI integration are planned next.

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
   [ ANALYZER ]   risk ✓ | policies ✓ | cost | ML score           (partial)
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
├── policies/            # OPA Rego rules (embedded into the binary)
│   ├── s3_public.rego
│   ├── sg_open.rego
│   ├── rds_encryption.rego
│   ├── iam_wildcard.rego
│   └── ebs_encryption.rego
├── internal/
│   ├── tfplan/          # plan JSON parsing and summary
│   ├── risk/            # SAFE → CRITICAL classification
│   └── policy/          # Finding + Checkov/tfsec/OPA scanners
└── testdata/            # fixture plans / scanner JSON for unit tests
```

## What works today

### `internal/tfplan`

| Piece | Responsibility |
|-------|----------------|
| `types.go` | Typed `Plan`, `ResourceChange`, `Change`; collapses Terraform action lists into a single `Action` |
| `parse.go` | `Parse` / `ParseFile`; returns `ErrNotAPlan` when `format_version` is missing |
| `summary.go` | Counts create/update/replace/delete; excludes no-ops and data sources |

### `internal/risk`

| Piece | Responsibility |
|-------|----------------|
| `Level` | `SAFE` → `CAUTION` → `DANGER` → `CRITICAL` |
| `Classify(action, resourceType)` | Base level from action; +1 if the resource is stateful |

### `internal/policy`

| Piece | Responsibility |
|-------|----------------|
| `Finding` | Unified issue shape shared by every scanner |
| `RunCheckov` / `RunTfsec` | Optional CLI wrappers (`os/exec` + JSON parse); **warn if binary missing** |
| `EvaluateOPA` | Runs embedded Rego policies via the OPA Go SDK |
| `Scan` | Merges Checkov + tfsec + OPA into one `Result` |

**Built-in OPA policies (`policies/`):**

| File | ID | What it catches |
|------|----|-----------------|
| `s3_public.rego` | `SENTINEL-S3-001` | S3 ACL public-read / public-read-write |
| `sg_open.rego` | `SENTINEL-SG-001` | Security group open to `0.0.0.0/0` |
| `rds_encryption.rego` | `SENTINEL-RDS-001` | RDS without `storage_encrypted` |
| `iam_wildcard.rego` | `SENTINEL-IAM-001` | IAM policy with `Action: "*"` |
| `ebs_encryption.rego` | `SENTINEL-EBS-001` | EBS volume without encryption |

## Getting started

### Prerequisites

- Go 1.26+
- Optional: [Checkov](https://www.checkov.io/) and/or [tfsec](https://github.com/aquasecurity/tfsec) on `PATH`

### Build and test

```bash
make test
# or
go test ./... -cover

go vet ./...
gofmt -l .
go build ./...
```

There is no CLI entrypoint yet. Exercise packages via unit tests and fixtures under `testdata/`.

### Example: classify risk

```go
level := risk.Classify(tfplan.ActionDelete, "aws_db_instance")
// level == risk.CRITICAL
```

### Example: run policies on a plan

```go
result, err := policy.Scan(ctx, plan, policy.ScanOptions{
    TerraformDir: "./infra", // optional; Checkov/tfsec scan this folder
})
// result.Findings → unified list
// result.Warnings → e.g. "checkov not found on PATH; skipping …"
```

## Roadmap

1. Terminal renderer + `scan` / `version` CLI
2. Wire risk + policies into the CLI with `--fail-on`
3. Static AWS cost delta estimation
4. Embedded logistic-regression risk score
5. Optional LLM explainer (Ollama / HTTP), disabled with `--no-ai`
6. GitHub Action (PR comment) and GoReleaser distribution

## Development notes

- Prefer table-driven tests and fixtures over live Terraform / live scanners in unit tests
- Wrap errors with `%w` and keep sentinel errors (`errors.Is`) for control flow
- Keep packages under `internal/` until a public API is intentionally exported
- Optional scanners must warn, never crash, when the binary is absent

## License

License file to be added (Apache 2.0 intended).
