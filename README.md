# IaC Sentinel

```text
  ┌─────────────────────────────────────────────────────────────┐
  │                                                             │
  │   ██╗ █████╗  ██████╗    ███████╗███████╗███╗   ██╗         │
  │   ██║██╔══██╗██╔════╝    ██╔════╝██╔════╝████╗  ██║         │
  │   ██║███████║██║         ███████╗█████╗  ██╔██╗ ██║         │
  │   ██║██╔══██║██║         ╚════██║██╔══╝  ██║╚██╗██║         │
  │   ██║██║  ██║╚██████╗    ███████║███████╗██║ ╚████║         │
  │   ╚═╝╚═╝  ╚═╝ ╚═════╝    ╚══════╝╚══════╝╚═╝  ╚═══╝         │
  │                                                             │
  │          Automated reviewer for Terraform plans             │
  └─────────────────────────────────────────────────────────────┘
```

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![AWS](https://img.shields.io/badge/Cloud-AWS-FF9900?style=for-the-badge&logo=amazon-aws&logoColor=white)](https://aws.amazon.com/)
[![OPA](https://img.shields.io/badge/Policy-OPA%20Rego-000000?style=for-the-badge&logo=openpolicyagent&logoColor=white)](https://www.openpolicyagent.org/)
[![Terraform](https://img.shields.io/badge/IaC-Terraform-844FBA?style=for-the-badge&logo=terraform&logoColor=white)](https://www.terraform.io/)
[![License](https://img.shields.io/badge/License-Apache%202.0%20(planned)-blue?style=for-the-badge)](#license)

> Catch risky Terraform changes **before** they merge — parse the plan, score the risk, enforce security policies.

| Status | Component |
|:------:|-----------|
| ✅ | Plan parser (`internal/tfplan`) |
| ✅ | Risk classifier (`internal/risk`) |
| ✅ | Policy engine (`internal/policy` + OPA + optional Checkov/tfsec) |
| 🚧 | CLI + terminal renderer + `--fail-on` |
| ⏳ | Cost estimation, ML score, LLM explainer, GitHub Action |

---

## Why this exists

```mermaid
flowchart LR
  A["👨‍💻 Dev opens PR"] --> B["CI: terraform plan"]
  B --> C{"Plan OK?"}
  C -->|yes| D["❌ Nobody reads<br/>400-line diff"]
  D --> E["💥 destroy RDS<br/>slips through"]
  C -->|fail| F["Blocked"]

  style E fill:#7f1d1d,stroke:#ef4444,color:#fff
  style D fill:#78350f,stroke:#f59e0b,color:#fff
```

**IaC Sentinel** sits after `terraform plan` and answers:

1. **What** changed? (create / update / replace / delete)
2. **How dangerous** is it? (`SAFE` → `CRITICAL`)
3. **Does it break** security policies? (OPA + optional scanners)

---

## High-level architecture

```mermaid
flowchart TB
  subgraph INPUT["📥 Input"]
    PLAN["terraform plan -json"]
    HCL["Terraform HCL sources"]
  end

  subgraph CORE["🧠 IaC Sentinel core"]
    PARSER["✅ PARSER<br/>internal/tfplan"]
    RISK["✅ RISK<br/>internal/risk"]
    POL["✅ POLICY<br/>internal/policy"]
    COST["⏳ COST"]
    ML["⏳ ML SCORE"]
  end

  subgraph SCANNERS["🔍 Optional scanners"]
    CKV["Checkov CLI"]
    TFS["tfsec CLI"]
    OPA["OPA Rego<br/>policies/*.rego"]
  end

  subgraph OUT["📤 Output (upcoming)"]
    CLI["Terminal table"]
    PR["PR markdown comment"]
    FAIL["--fail-on CRITICAL"]
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

  style PARSER fill:#14532d,stroke:#22c55e,color:#fff
  style RISK fill:#14532d,stroke:#22c55e,color:#fff
  style POL fill:#14532d,stroke:#22c55e,color:#fff
  style COST fill:#1e3a5f,stroke:#64748b,color:#fff
  style ML fill:#1e3a5f,stroke:#64748b,color:#fff
```

---

## Data flow (what happens to one change)

```mermaid
sequenceDiagram
  participant TF as Terraform
  participant P as tfplan.Parse
  participant R as risk.Classify
  participant O as policy.Scan
  participant U as Future CLI

  TF->>P: plan.json
  P->>P: ResourceChange + Action
  P->>R: action + resource type
  R-->>U: Level (SAFE…CRITICAL)
  P->>O: plan input
  O->>O: OPA Rego + Checkov/tfsec
  O-->>U: []Finding + Warnings
  U-->>U: print / fail CI
```

---

## Repository map

```text
iac-sentinel/
│
├── 📄 go.mod / go.sum          Go module + dependencies (OPA SDK, …)
├── 🛠️  Makefile                 make test | build | fmt | vet
├── 📖 README.md                 you are here
│
├── 📁 policies/                 OPA rules (embedded in the binary)
│   ├── s3_public.rego
│   ├── sg_open.rego
│   ├── rds_encryption.rego
│   ├── iam_wildcard.rego
│   └── ebs_encryption.rego
│
├── 📁 internal/                 private application packages
│   ├── tfplan/                  ① parse plan JSON
│   ├── risk/                    ② score danger
│   └── policy/                  ③ security findings
│
└── 📁 testdata/                 fixtures for unit tests (no live AWS needed)
```

```mermaid
mindmap
  root((IaC Sentinel))
    tfplan
      Parse / ParseFile
      Action collapse
      Summarize counts
    risk
      Level iota
      Stateful table
      Classify + escalate
    policy
      Finding unified
      Checkov wrapper
      tfsec wrapper
      OPA Evaluate
      Scan merge
    policies Rego
      S3 public
      SG 0.0.0.0/0
      RDS encrypt
      IAM wildcard
      EBS encrypt
```

---

## Package deep-dive

### ① `internal/tfplan` — understand the plan

| File | Role |
|------|------|
| `types.go` | `Plan`, `ResourceChange`, `Change`, `Action` |
| `parse.go` | Decode JSON → Go structs (`ErrNotAPlan` if invalid) |
| `summary.go` | Count creates/updates/replaces/deletes (skip no-ops & data sources) |

```mermaid
flowchart LR
  J["plan.json"] --> PARSE["ParseFile"]
  PARSE --> PLAN["Plan"]
  PLAN --> ACT["Change.Action()"]
  ACT --> S["Summarize"]
  S --> OUT["Creates / Updates<br/>Replaces / Deletes"]
```

Terraform sometimes encodes a **replace** as `["delete","create"]`.  
`Change.Action()` collapses that into a single `replace`.

---

### ② `internal/risk` — how dangerous is it?

```mermaid
flowchart TD
  A["Action + resource type"] --> B{"baseLevel(action)"}
  B -->|create / no-op / read| S["SAFE"]
  B -->|update| C["CAUTION"]
  B -->|replace / delete| D["DANGER"]
  S --> E{"IsStateful?"}
  C --> E
  D --> E
  E -->|no| F["return level"]
  E -->|yes| G["escalate +1<br/>(cap CRITICAL)"]
  G --> F
```

| Action (non-stateful) | Level |
|-----------------------|-------|
| create / no-op / read | `SAFE` |
| update | `CAUTION` |
| replace / delete | `DANGER` |

**Stateful resources** (RDS, S3, EBS, DynamoDB, …) bump +1:

| Example | Result |
|---------|--------|
| create S3 bucket | `CAUTION` |
| update RDS | `DANGER` |
| **delete RDS** | **`CRITICAL`** |

```text
  SAFE ──► CAUTION ──► DANGER ──► CRITICAL
   0         1           2           3
              ▲ escalate if stateful ▲
```

---

### ③ `internal/policy` — security findings

```mermaid
flowchart TB
  subgraph SOURCES["Three sources → one shape"]
    CKV["Checkov<br/>(optional CLI)"]
    TFS["tfsec<br/>(optional CLI)"]
    OPA["OPA Rego<br/>(always in-process)"]
  end

  CKV --> F["Finding"]
  TFS --> F
  OPA --> F
  F --> R["Result{Findings, Warnings}"]

  CKV -.->|binary missing| W["⚠️ Warning — do not crash"]
  TFS -.->|binary missing| W
```

**Unified `Finding` fields:** `ID`, `Source`, `Severity`, `Title`, `Description`, `Resource`, `File`, `Guideline`.

#### Built-in OPA policies

| Badge | File | ID | Catches |
|:-----:|------|----|---------|
| 🪣 | `s3_public.rego` | `SENTINEL-S3-001` | Public S3 ACL |
| 🔓 | `sg_open.rego` | `SENTINEL-SG-001` | SG open to `0.0.0.0/0` |
| 🗄️ | `rds_encryption.rego` | `SENTINEL-RDS-001` | RDS not encrypted |
| 🔑 | `iam_wildcard.rego` | `SENTINEL-IAM-001` | IAM `Action: "*"` |
| 💾 | `ebs_encryption.rego` | `SENTINEL-EBS-001` | EBS not encrypted |

---

## Quick start

### Prerequisites

- **Go 1.26+**
- Optional on `PATH`: [Checkov](https://www.checkov.io/), [tfsec](https://github.com/aquasecurity/tfsec)

### Commands

```bash
make test          # go test ./... -cover
make vet           # go vet ./...
make fmt           # go fmt ./...
make build         # binary → bin/iac-sentinel
```

> There is **no CLI entrypoint yet**. Exercise the libraries via unit tests and `testdata/` fixtures.

### Snippets

**Risk**

```go
level := risk.Classify(tfplan.ActionDelete, "aws_db_instance")
fmt.Println(level.String()) // CRITICAL
```

**Policies**

```go
result, err := policy.Scan(ctx, plan, policy.ScanOptions{
    TerraformDir: "./infra", // Checkov/tfsec; skipped with a warning if missing
})
for _, f := range result.Findings {
    fmt.Printf("[%s] %s — %s\n", f.Severity, f.ID, f.Title)
}
for _, w := range result.Warnings {
    fmt.Println("warn:", w)
}
```

---

## Roadmap

```mermaid
timeline
  title IaC Sentinel delivery path
  section Done
    Parser + Risk + Policies : tfplan / risk / policy + Rego
  section Next
    CLI scan / version : terminal renderer + --fail-on
  section Later
    Cost + ML : AWS delta + logistic risk score
    Explainer : optional LLM (--no-ai supported)
    Ship : GitHub Action + GoReleaser
```

1. ✅ Plan parser  
2. ✅ Risk classification  
3. ✅ Policy engine (OPA + Checkov/tfsec wrappers)  
4. 🚧 Terminal renderer + `scan` / `version` CLI + `--fail-on`  
5. ⏳ Static AWS cost delta  
6. ⏳ Embedded logistic-regression risk score  
7. ⏳ Optional LLM explainer (`--no-ai`)  
8. ⏳ GitHub Action (PR comment) + GoReleaser  

---

## Development notes

- Prefer **table-driven tests** + fixtures over live Terraform / live scanners  
- Wrap errors with `%w`; use sentinel errors + `errors.Is`  
- Code under `internal/` stays private until a public API is intentional  
- Optional scanners **warn**, never crash, when the binary is absent  

---

## License

Apache 2.0 intended (license file to be added).
