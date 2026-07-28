# tfguard

**Langue :** [English](README.md) | Français

Revue de plans Terraform (AWS) : parse du JSON, classification de risque (`SAFE` → `CRITICAL`), policies OPA (et Checkov/tfsec optionnels), échec CI via `-fail-on`.

## Pipeline

```mermaid
flowchart LR
  A[plan.json] --> B[tfplan]
  B --> C[risk]
  B --> D[policy]
  C --> E[app.Report]
  D --> E
  E --> F[CLI]
```

| Couche | Package | Role |
|--------|---------|------|
| Parse | `internal/tfplan` | Lire le plan JSON |
| Risque | `internal/risk` | Noter chaque changement |
| Policy | `internal/policy` | OPA + scanners optionnels |
| Orchestration | `internal/app` | Rapport + fail-on |
| CLI | `cmd/tfguard` | Commandes Cobra : `scan`, `version` |

## Usage

```bash
make build
./bin/tfguard scan --plan plan.json
./bin/tfguard scan --plan plan.json --dir ./infra --fail-on DANGER
```

Codes de sortie : `0` ok · `1` seuil / erreur · `2` usage.

## Risque

create → `SAFE` · update → `CAUTION` · replace/delete → `DANGER`  
Stateful (RDS, S3, EBS…) → +1 (max `CRITICAL`).

## Roadmap

Fait : parse, risk, policies, CLI.  
Suivant : estimation de coût AWS.  
Prévu : score ML, explainer LLM, GitHub Action.

## Licence

Apache 2.0 prévu.
