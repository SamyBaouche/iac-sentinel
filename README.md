# IaC Sentinel

Automated reviewer for Terraform plans. IaC Sentinel reads `terraform plan -json` output and surfaces destructive changes, policy violations, and (planned) cost impact so teams do not ship risky infrastructure changes unnoticed.

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Terraform](https://img.shields.io/badge/IaC-Terraform-844FBA?logo=terraform&logoColor=white)](https://www.terraform.io/)
[![OPA](https://img.shields.io/badge/Policy-OPA%20Rego-000000?logo=openpolicyagent&logoColor=white)](https://www.openpolicyagent.org/)
[![AWS](https://img.shields.io/badge/Cloud-AWS-FF9900?logo=amazon-aws&logoColor=white)](https://aws.amazon.com/)

| Status | Component |
|--------|-----------|
| Done | Plan parser (`internal/tfplan`) |
| Done | Risk classifier (`internal/risk`) |
| Done | Policy engine (`internal/policy`, OPA Rego, optional Checkov/tfsec) |
| Next | CLI, terminal renderer, `--fail-on` |
| Planned | Cost estimation, ML score, LLM explainer, GitHub Action |

---

## Problem

CI often stops at “terraform plan succeeded.” Nobody reads a long plan diff. A single `destroy` on a production database can slip through. IaC Sentinel is meant to catch that before merge.

```mermaid
flowchart LR
  A[Developer opens PR] --> B[CI runs terraform plan]
  B --> C{Plan succeeds?}
  C -->|yes| D[Diff rarely reviewed in full]
  D --> E[Destructive change may merge]
  C -->|no| F[Pipeline blocked]

  style E fill:#3f1d1d,stroke:#b91c1c,color:#fff
```

---

## Scope (v1)

- AWS only
- Usable without any AI component (`--no-ai` when the explainer lands)
- Optional external scanners (Checkov, tfsec): warn if missing, never hard-fail solely because a binary is absent
- Idiomatic Go: wrapped errors, table-driven tests, `gofmt` / `go vet` clean

---

## Architecture

```mermaid
flowchart TB
  subgraph input [Input]
    PLAN[terraform plan -json]
    HCL[Terraform HCL sources]
  end

  subgraph core [IaC Sentinel]
    PARSER[PARSER - internal/tfplan]
    RISK[RISK - internal/risk]
    POL[POLICY - internal/policy]
    COST[COST - planned]
    ML[ML SCORE - planned]
  end

  subgraph scanners [Scanners]
    CKV[Checkov CLI - optional]
    TFS[tfsec CLI - optional]
    OPA[OPA Rego - embedded]
  end

  subgraph output [Output - planned]
    CLI[Terminal table]
    PR[PR markdown comment]
    FAIL["--fail-on threshold"]
  end

  PLAN --> PARSER
  PARSER --> RISK
  PARSER --> POL
  HCL --> CKV
  HCL --> TFS
  PARSER --> OPA
  CKV --> POL
  TFS --> POL
  OPA --> POL
  RISK --> CLI
  POL --> CLI
  RISK --> FAIL
  POL --> FAIL
  CLI --> PR
```

### Request flow

```mermaid
sequenceDiagram
  participant TF as Terraform
  participant P as tfplan.Parse
  participant R as risk.Classify
  participant O as policy.Scan
  participant U as CLI planned

  TF->>P: plan.json
  P->>P: ResourceChange and Action
  P->>R: action + resource type
  R-->>U: Level SAFE to CRITICAL
  P->>O: plan input
  O->>O: OPA plus optional Checkov/tfsec
  O-->>U: Findings and Warnings
```

---

## Repository layout

```text
iac-sentinel/
├── go.mod / go.sum
├── Makefile
├── README.md
├── policies/                 # OPA Rego rules (embedded in the binary)
│   ├── s3_public.rego
│   ├── sg_open.rego
│   ├── rds_encryption.rego
│   ├── iam_wildcard.rego
│   └── ebs_encryption.rego
├── internal/
│   ├── tfplan/               # plan JSON parsing and summary
│   ├── risk/                 # SAFE → CRITICAL classification
│   └── policy/               # Finding + Checkov / tfsec / OPA
└── testdata/                 # fixtures for unit tests
```

---

## What works today

### `internal/tfplan`

| Piece | Responsibility |
|-------|----------------|
| `types.go` | `Plan`, `ResourceChange`, `Change`, `Action`; collapses replace encoded as `["delete","create"]` |
| `parse.go` | `Parse` / `ParseFile`; returns `ErrNotAPlan` when `format_version` is missing |
| `summary.go` | Counts create/update/replace/delete; excludes no-ops and data sources |

```mermaid
flowchart LR
  J[plan.json] --> PARSE[ParseFile]
  PARSE --> PLAN[Plan]
  PLAN --> ACT[Change.Action]
  ACT --> S[Summarize]
  S --> OUT[Creates Updates Replaces Deletes]
```

### `internal/risk`

| Piece | Responsibility |
|-------|----------------|
| `Level` | `SAFE` → `CAUTION` → `DANGER` → `CRITICAL` (`iota`), plus `String()` |
| Stateful table | AWS types that hold durable data (RDS, S3, EBS, DynamoDB, …) |
| `Classify(action, resourceType)` | Base level from action; escalate +1 when stateful (capped at `CRITICAL`) |

```mermaid
flowchart TD
  A[Action + resource type] --> B{baseLevel}
  B -->|create / no-op / read| S[SAFE]
  B -->|update| C[CAUTION]
  B -->|replace / delete| D[DANGER]
  S --> E{IsStateful?}
  C --> E
  D --> E
  E -->|no| F[return level]
  E -->|yes| G[escalate +1 capped at CRITICAL]
  G --> F
```

**Base mapping (non-stateful)**

| Action | Level |
|--------|-------|
| create / no-op / read / unknown | `SAFE` |
| update | `CAUTION` |
| replace / delete | `DANGER` |

**Examples with escalation:** create S3 → `CAUTION`; update RDS → `DANGER`; delete RDS → `CRITICAL`.

### `internal/policy`

| Piece | Responsibility |
|-------|----------------|
| `Finding` | Unified issue shape shared by every scanner |
| `RunCheckov` / `RunTfsec` | Optional CLI wrappers (`os/exec` + JSON); warn if binary missing |
| `EvaluateOPA` | Embedded Rego policies via the OPA Go SDK |
| `Scan` | Merges Checkov + tfsec + OPA into one `Result` |

```mermaid
flowchart TB
  CKV[Checkov optional] --> F[Finding]
  TFS[tfsec optional] --> F
  OPA[OPA Rego] --> F
  F --> R["Result Findings + Warnings"]
  CKV -.->|missing binary| W[Warning only]
  TFS -.->|missing binary| W
```

**Built-in OPA policies**

| File | ID | Catches |
|------|----|---------|
| `s3_public.rego` | `SENTINEL-S3-001` | S3 ACL public-read / public-read-write |
| `sg_open.rego` | `SENTINEL-SG-001` | Security group open to `0.0.0.0/0` |
| `rds_encryption.rego` | `SENTINEL-RDS-001` | RDS without `storage_encrypted` |
| `iam_wildcard.rego` | `SENTINEL-IAM-001` | IAM policy with `Action: "*"` |
| `ebs_encryption.rego` | `SENTINEL-EBS-001` | EBS volume without encryption |

---

## Getting started

### Prerequisites

- Go 1.26+
- Optional: [Checkov](https://www.checkov.io/) and/or [tfsec](https://github.com/aquasecurity/tfsec) on `PATH`

### Build and test

```bash
make test
make vet
make fmt
make build
```

There is no CLI entrypoint yet. Exercise packages via unit tests and fixtures under `testdata/`.

### Examples

```go
level := risk.Classify(tfplan.ActionDelete, "aws_db_instance")
// level == risk.CRITICAL
```

```go
result, err := policy.Scan(ctx, plan, policy.ScanOptions{
    TerraformDir: "./infra",
})
// result.Findings — unified list
// result.Warnings — e.g. checkov not found; skipping
```

---

## Roadmap

```mermaid
timeline
  title Delivery path
  section Completed
    Parser Risk Policies : tfplan risk policy and Rego
  section In progress
    CLI : scan version renderer fail-on
  section Planned
    Cost and ML : AWS delta and logistic score
    Explainer and ship : optional LLM GitHub Action GoReleaser
```

1. Plan parser — done
2. Risk classification — done
3. Policy engine (OPA + Checkov/tfsec wrappers) — done
4. Terminal renderer + `scan` / `version` CLI + `--fail-on`
5. Static AWS cost delta estimation
6. Embedded logistic-regression risk score
7. Optional LLM explainer (`--no-ai`)
8. GitHub Action (PR comment) and GoReleaser distribution

---

## Development notes

- Prefer table-driven tests and fixtures over live Terraform or live scanners
- Wrap errors with `%w`; use sentinel errors with `errors.Is`
- Keep packages under `internal/` until a public API is intentional
- Optional scanners must warn, never crash, when the binary is absent

---

## License

Apache 2.0 intended (license file to be added).
