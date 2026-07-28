# IaC Sentinel

**Language:** English | [Français](README.fr.md)

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
| Done | CLI (`scan` / `version`), terminal renderer, `--fail-on` |
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

## Project structure

### Directory tree

```mermaid
flowchart TB
  ROOT[iac-sentinel]

  ROOT --> GOMOD[go.mod / go.sum]
  ROOT --> MAKE[Makefile]
  ROOT --> README[README.md / README.fr.md]
  ROOT --> CMD[cmd/iac-sentinel/]
  ROOT --> POLDIR[policies/]
  ROOT --> INTERNAL[internal/]
  ROOT --> TESTDATA[testdata/]

  CMD --> MAIN[main.go]

  POLDIR --> P1[s3_public.rego]
  POLDIR --> P2[sg_open.rego]
  POLDIR --> P3[rds_encryption.rego]
  POLDIR --> P4[iam_wildcard.rego]
  POLDIR --> P5[ebs_encryption.rego]
  POLDIR --> PEMB[embed.go]

  INTERNAL --> TF[tfplan/]
  INTERNAL --> RK[risk/]
  INTERNAL --> PY[policy/]
  INTERNAL --> APP[app/]
  INTERNAL --> REN[render/]

  TF --> TF1[types.go]
  TF --> TF2[parse.go]
  TF --> TF3[summary.go]
  TF --> TF4[tfplan_test.go]

  RK --> RK1[risk.go]
  RK --> RK2[risk_test.go]

  PY --> PY1[finding.go]
  PY --> PY2[checkov.go]
  PY --> PY3[tfsec.go]
  PY --> PY4[opa.go]
  PY --> PY5[scan.go]
  PY --> PY6[policy_test.go]

  TESTDATA --> T1[plan_*.json]
  TESTDATA --> T2[checkov_failed.json]
  TESTDATA --> T3[tfsec_failed.json]

  style ROOT fill:#0f172a,stroke:#38bdf8,color:#fff
  style INTERNAL fill:#1e293b,stroke:#94a3b8,color:#fff
  style POLDIR fill:#1e293b,stroke:#94a3b8,color:#fff
  style TESTDATA fill:#1e293b,stroke:#94a3b8,color:#fff
```

### Package responsibilities

```mermaid
flowchart LR
  subgraph packages [Go packages]
    TFPLAN[internal/tfplan<br/>parse plan JSON]
    RISK[internal/risk<br/>score danger]
    POLICY[internal/policy<br/>security findings]
    POLICIES[policies<br/>Rego rules + embed]
  end

  TFPLAN -->|Action + Plan| RISK
  TFPLAN -->|PlanInput| POLICY
  POLICIES -->|embedded .rego| POLICY

  style TFPLAN fill:#14532d,stroke:#22c55e,color:#fff
  style RISK fill:#1e3a8a,stroke:#60a5fa,color:#fff
  style POLICY fill:#713f12,stroke:#f59e0b,color:#fff
  style POLICIES fill:#4c1d95,stroke:#a78bfa,color:#fff
```

### Dependency graph

```mermaid
flowchart TB
  CLI["cmd/iac-sentinel"]
  APP[internal/app]
  RENDER[internal/render]
  RISK[internal/risk]
  POLICY[internal/policy]
  TFPLAN[internal/tfplan]
  POLICIES[policies]
  OPA[OPA Go SDK]
  EXT[Checkov / tfsec CLIs]

  CLI --> APP
  CLI --> RENDER
  RENDER --> APP
  APP --> RISK
  APP --> POLICY
  APP --> TFPLAN
  RISK --> TFPLAN
  POLICY --> TFPLAN
  POLICY --> POLICIES
  POLICY --> OPA
  POLICY -.->|os/exec optional| EXT
```

### Text layout

```text
iac-sentinel/
├── go.mod / go.sum
├── Makefile
├── README.md
├── README.fr.md
├── cmd/iac-sentinel/         CLI entrypoint (scan, version)
│   ├── main.go
│   └── main_test.go
├── policies/                 OPA Rego rules (embedded in the binary)
│   ├── embed.go
│   ├── s3_public.rego
│   ├── sg_open.rego
│   ├── rds_encryption.rego
│   ├── iam_wildcard.rego
│   └── ebs_encryption.rego
├── internal/
│   ├── tfplan/               1. parse plan JSON
│   ├── risk/                 2. SAFE → CRITICAL
│   ├── policy/               3. Finding + scanners
│   ├── app/                  4. orchestrate scan + --fail-on
│   └── render/               5. terminal report
└── testdata/                 fixtures for unit tests
```

---

## System architecture

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

### End-to-end sequence

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

## Module details

### `internal/tfplan` — understand the plan

| File | Responsibility |
|------|----------------|
| `types.go` | `Plan`, `ResourceChange`, `Change`, `Action`; collapses replace encoded as `["delete","create"]` |
| `parse.go` | `Parse` / `ParseFile`; returns `ErrNotAPlan` when `format_version` is missing |
| `summary.go` | Counts create/update/replace/delete; excludes no-ops and data sources |
| `tfplan_test.go` | Fixture-based unit tests |

```mermaid
flowchart LR
  J[plan.json] --> PARSE[ParseFile]
  PARSE --> PLAN[Plan]
  PLAN --> ACT[Change.Action]
  ACT --> S[Summarize]
  S --> OUT[Creates Updates Replaces Deletes]
```

```mermaid
classDiagram
  class Plan {
    FormatVersion string
    TerraformVersion string
    ResourceChanges ResourceChange
  }
  class ResourceChange {
    Address string
    Mode string
    Type string
    Name string
    Change Change
  }
  class Change {
    Actions string
    Before RawMessage
    After RawMessage
    Action() Action
  }
  class Action {
    <<enumeration>>
    no-op
    create
    read
    update
    delete
    replace
  }
  Plan "1" --> "*" ResourceChange
  ResourceChange "1" --> "1" Change
  Change --> Action
```

---

### `internal/risk` — how dangerous is it?

| File | Responsibility |
|------|----------------|
| `risk.go` | `Level`, stateful table, `Classify`, `String` |
| `risk_test.go` | Covers base levels and stateful escalation |

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

```mermaid
stateDiagram-v2
  [*] --> SAFE: create / no-op / read
  [*] --> CAUTION: update
  [*] --> DANGER: replace / delete

  SAFE --> CAUTION: stateful +1
  CAUTION --> DANGER: stateful +1
  DANGER --> CRITICAL: stateful +1
  CRITICAL --> CRITICAL: already max
```

**Base mapping (non-stateful)**

| Action | Level |
|--------|-------|
| create / no-op / read / unknown | `SAFE` |
| update | `CAUTION` |
| replace / delete | `DANGER` |

**Examples with escalation:** create S3 → `CAUTION`; update RDS → `DANGER`; delete RDS → `CRITICAL`.

---

### `internal/policy` — security findings

| File | Responsibility |
|------|----------------|
| `finding.go` | Unified `Finding` + `Result` |
| `checkov.go` | Checkov CLI wrapper (`os/exec` + JSON) |
| `tfsec.go` | tfsec CLI wrapper |
| `opa.go` | OPA SDK evaluation + `PlanInput` |
| `scan.go` | Merge all scanners |
| `policy_test.go` | JSON fixtures + OPA cases + missing-binary warnings |

```mermaid
flowchart TB
  CKV[Checkov optional] --> F[Finding]
  TFS[tfsec optional] --> F
  OPA[OPA Rego] --> F
  F --> R[Result Findings + Warnings]
  CKV -.->|missing binary| W[Warning only]
  TFS -.->|missing binary| W
```

```mermaid
flowchart LR
  subgraph rego [policies/*.rego]
    S3[SENTINEL-S3-001]
    SG[SENTINEL-SG-001]
    RDS[SENTINEL-RDS-001]
    IAM[SENTINEL-IAM-001]
    EBS[SENTINEL-EBS-001]
  end

  EMBED[embed.go] --> FS[embedded FS]
  FS --> OPA[EvaluateOPA]
  rego --> EMBED
  PLAN[PlanInput from tfplan] --> OPA
  OPA --> FINDINGS[Finding list]
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
make build
./bin/iac-sentinel version
```

### Scan a plan

```bash
# Analyze a terraform plan JSON file
./bin/iac-sentinel scan -plan testdata/plan_mixed.json

# Also run Checkov/tfsec against Terraform sources (warn if missing)
./bin/iac-sentinel scan -plan plan.json -dir ./infra

# Fail CI when risk or findings reach DANGER or above
./bin/iac-sentinel scan -plan plan.json -fail-on DANGER
echo $?   # 1 if threshold triggered, 0 otherwise
```

| Flag | Meaning |
|------|---------|
| `-plan` | Path to `terraform plan -json` output (required) |
| `-dir` | Terraform HCL directory for Checkov/tfsec |
| `-fail-on` | `SAFE` / `CAUTION` / `DANGER` / `CRITICAL` — exit 1 if reached |
| `-skip-checkov` / `-skip-tfsec` / `-skip-opa` | Disable individual scanners |

### Library examples

```go
level := risk.Classify(tfplan.ActionDelete, "aws_db_instance")
// level == risk.CRITICAL
```

```go
result, err := policy.Scan(ctx, plan, policy.ScanOptions{
    TerraformDir: "./infra",
})
```

---

## Roadmap

```mermaid
timeline
  title Delivery path
  section Completed
    Parser Risk Policies CLI : tfplan risk policy Rego scan fail-on
  section Planned
    Cost and ML : AWS delta and logistic score
    Explainer and ship : optional LLM GitHub Action GoReleaser
```

1. Plan parser — done
2. Risk classification — done
3. Policy engine (OPA + Checkov/tfsec wrappers) — done
4. Terminal renderer + `scan` / `version` CLI + `--fail-on` — done
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
