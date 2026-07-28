# IaC Sentinel

**Langue :** [English](README.md) | Français

Revueur automatique de plans Terraform. IaC Sentinel lit la sortie `terraform plan -json`, met en évidence les changements destructeurs, les violations de politiques et (à venir) l’impact coût, afin d’éviter que des modifications d’infrastructure risquées passent inaperçues.

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Terraform](https://img.shields.io/badge/IaC-Terraform-844FBA?logo=terraform&logoColor=white)](https://www.terraform.io/)
[![OPA](https://img.shields.io/badge/Policy-OPA%20Rego-000000?logo=openpolicyagent&logoColor=white)](https://www.openpolicyagent.org/)
[![AWS](https://img.shields.io/badge/Cloud-AWS-FF9900?logo=amazon-aws&logoColor=white)](https://aws.amazon.com/)

| Statut | Composant |
|--------|-----------|
| Fait | Parseur de plan (`internal/tfplan`) |
| Fait | Classificateur de risque (`internal/risk`) |
| Fait | Moteur de policies (`internal/policy`, OPA Rego, Checkov/tfsec optionnels) |
| Fait | CLI (`scan` / `version`), rendu terminal, `--fail-on` |
| Prévu | Estimation de coût, score ML, explainer LLM, GitHub Action |

---

## Problème

La CI s’arrête souvent à « terraform plan succeeded ». Personne ne lit un diff de plan de plusieurs centaines de lignes. Un seul `destroy` sur une base de production peut passer. IaC Sentinel vise à le détecter avant le merge.

```mermaid
flowchart LR
  A[Le developpeur ouvre une PR] --> B[CI: terraform plan]
  B --> C{Plan OK?}
  C -->|oui| D[Le diff est rarement lu en entier]
  D --> E[Un changement destructeur peut merger]
  C -->|non| F[Pipeline bloque]

  style E fill:#3f1d1d,stroke:#b91c1c,color:#fff
```

---

## Perimetre (v1)

- AWS uniquement
- Utilisable sans composant IA (`--no-ai` lorsque l’explainer arrivera)
- Scanners externes optionnels (Checkov, tfsec) : avertir s’ils manquent, ne jamais faire echouer uniquement pour un binaire absent
- Go idiomatique : erreurs wrappees, tests table-driven, `gofmt` / `go vet` propres

---

## Structure du projet

### Arborescence

```mermaid
flowchart TB
  ROOT[iac-sentinel]

  ROOT --> GOMOD[go.mod / go.sum]
  ROOT --> MAKE[Makefile]
  ROOT --> README[README.md / README.fr.md]
  ROOT --> POLDIR[policies/]
  ROOT --> INTERNAL[internal/]
  ROOT --> TESTDATA[testdata/]

  POLDIR --> P1[s3_public.rego]
  POLDIR --> P2[sg_open.rego]
  POLDIR --> P3[rds_encryption.rego]
  POLDIR --> P4[iam_wildcard.rego]
  POLDIR --> P5[ebs_encryption.rego]
  POLDIR --> PEMB[embed.go]

  INTERNAL --> TF[tfplan/]
  INTERNAL --> RK[risk/]
  INTERNAL --> PY[policy/]

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

### Roles des packages

```mermaid
flowchart LR
  subgraph packages [Packages Go]
    TFPLAN[internal/tfplan<br/>parse le plan JSON]
    RISK[internal/risk<br/>note le danger]
    POLICY[internal/policy<br/>constats de securite]
    POLICIES[policies<br/>regles Rego + embed]
  end

  TFPLAN -->|Action + Plan| RISK
  TFPLAN -->|PlanInput| POLICY
  POLICIES -->|rego embarque| POLICY

  style TFPLAN fill:#14532d,stroke:#22c55e,color:#fff
  style RISK fill:#1e3a8a,stroke:#60a5fa,color:#fff
  style POLICY fill:#713f12,stroke:#f59e0b,color:#fff
  style POLICIES fill:#4c1d95,stroke:#a78bfa,color:#fff
```

### Graphe de dependances

```mermaid
flowchart TB
  CLI["cmd/ CLI - prevu"]
  RISK[internal/risk]
  POLICY[internal/policy]
  TFPLAN[internal/tfplan]
  POLICIES[policies]
  OPA[OPA Go SDK]
  EXT[Checkov / tfsec CLIs]

  CLI -.->|futur| RISK
  CLI -.->|futur| POLICY
  CLI -.->|futur| TFPLAN

  RISK --> TFPLAN
  POLICY --> TFPLAN
  POLICY --> POLICIES
  POLICY --> OPA
  POLICY -.->|os/exec optionnel| EXT

  style CLI fill:#334155,stroke:#64748b,color:#fff,stroke-dasharray: 5 5
```

### Layout texte

```text
iac-sentinel/
├── go.mod / go.sum
├── Makefile
├── README.md / README.fr.md
├── policies/                 regles OPA Rego (embarquees dans le binaire)
│   ├── embed.go
│   ├── s3_public.rego
│   ├── sg_open.rego
│   ├── rds_encryption.rego
│   ├── iam_wildcard.rego
│   └── ebs_encryption.rego
├── internal/
│   ├── tfplan/               1. parser le plan JSON
│   │   ├── types.go
│   │   ├── parse.go
│   │   ├── summary.go
│   │   └── tfplan_test.go
│   ├── risk/                 2. SAFE → CRITICAL
│   │   ├── risk.go
│   │   └── risk_test.go
│   └── policy/               3. Finding + scanners
│       ├── finding.go
│       ├── checkov.go
│       ├── tfsec.go
│       ├── opa.go
│       ├── scan.go
│       └── policy_test.go
└── testdata/                 fixtures pour les tests unitaires
    ├── plan_minimal.json
    ├── plan_mixed.json
    ├── plan_replace.json
    ├── plan_not_a_plan.json
    ├── checkov_failed.json
    └── tfsec_failed.json
```

---

## Architecture systeme

```mermaid
flowchart TB
  subgraph input [Entree]
    PLAN[terraform plan -json]
    HCL[Sources Terraform HCL]
  end

  subgraph core [IaC Sentinel]
    PARSER[PARSER - internal/tfplan]
    RISK[RISK - internal/risk]
    POL[POLICY - internal/policy]
    COST[COST - prevu]
    ML[ML SCORE - prevu]
  end

  subgraph scanners [Scanners]
    CKV[Checkov CLI - optionnel]
    TFS[tfsec CLI - optionnel]
    OPA[OPA Rego - embarque]
  end

  subgraph output [Sortie - prevue]
    CLI[Tableau terminal]
    PR[Commentaire PR markdown]
    FAIL["seuil --fail-on"]
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

### Sequence de bout en bout

```mermaid
sequenceDiagram
  participant TF as Terraform
  participant P as tfplan.Parse
  participant R as risk.Classify
  participant O as policy.Scan
  participant U as CLI prevue

  TF->>P: plan.json
  P->>P: ResourceChange et Action
  P->>R: action + type de ressource
  R-->>U: Level SAFE a CRITICAL
  P->>O: entree plan
  O->>O: OPA plus Checkov/tfsec optionnels
  O-->>U: Findings et Warnings
```

---

## Detail des modules

### `internal/tfplan` — comprendre le plan

| Fichier | Responsabilite |
|---------|----------------|
| `types.go` | `Plan`, `ResourceChange`, `Change`, `Action` ; regroupe un replace encode en `["delete","create"]` |
| `parse.go` | `Parse` / `ParseFile` ; retourne `ErrNotAPlan` si `format_version` manque |
| `summary.go` | Compte create/update/replace/delete ; exclut no-ops et data sources |
| `tfplan_test.go` | Tests unitaires sur fixtures |

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

### `internal/risk` — quel est le niveau de danger ?

| Fichier | Responsabilite |
|---------|----------------|
| `risk.go` | `Level`, table stateful, `Classify`, `String` |
| `risk_test.go` | Couvre niveaux de base et escalade stateful |

```mermaid
flowchart TD
  A[Action + type de ressource] --> B{baseLevel}
  B -->|create / no-op / read| S[SAFE]
  B -->|update| C[CAUTION]
  B -->|replace / delete| D[DANGER]
  S --> E{IsStateful?}
  C --> E
  D --> E
  E -->|non| F[retourner le niveau]
  E -->|oui| G[escalade +1 plafonnee a CRITICAL]
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
  CRITICAL --> CRITICAL: deja au max
```

**Mapping de base (non stateful)**

| Action | Niveau |
|--------|--------|
| create / no-op / read / inconnu | `SAFE` |
| update | `CAUTION` |
| replace / delete | `DANGER` |

**Exemples avec escalade :** create S3 → `CAUTION` ; update RDS → `DANGER` ; delete RDS → `CRITICAL`.

---

### `internal/policy` — constats de securite

| Fichier | Responsabilite |
|---------|----------------|
| `finding.go` | `Finding` unifie + `Result` |
| `checkov.go` | Wrapper CLI Checkov (`os/exec` + JSON) |
| `tfsec.go` | Wrapper CLI tfsec |
| `opa.go` | Evaluation SDK OPA + `PlanInput` |
| `scan.go` | Fusion de tous les scanners |
| `policy_test.go` | Fixtures JSON + cas OPA + warning si binaire absent |

```mermaid
flowchart TB
  CKV[Checkov optionnel] --> F[Finding]
  TFS[tfsec optionnel] --> F
  OPA[OPA Rego] --> F
  F --> R[Result Findings + Warnings]
  CKV -.->|binaire absent| W[Warning seulement]
  TFS -.->|binaire absent| W
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

  EMBED[embed.go] --> FS[FS embarque]
  FS --> OPA[EvaluateOPA]
  rego --> EMBED
  PLAN[PlanInput depuis tfplan] --> OPA
  OPA --> FINDINGS[liste de Finding]
```

**Policies OPA integrees**

| Fichier | ID | Detecte |
|---------|----|---------|
| `s3_public.rego` | `SENTINEL-S3-001` | ACL S3 public-read / public-read-write |
| `sg_open.rego` | `SENTINEL-SG-001` | Security group ouvert a `0.0.0.0/0` |
| `rds_encryption.rego` | `SENTINEL-RDS-001` | RDS sans `storage_encrypted` |
| `iam_wildcard.rego` | `SENTINEL-IAM-001` | Policy IAM avec `Action: "*"` |
| `ebs_encryption.rego` | `SENTINEL-EBS-001` | Volume EBS non chiffre |

---

## Demarrage rapide

### Prerequis

- Go 1.26+
- Optionnel : [Checkov](https://www.checkov.io/) et/ou [tfsec](https://github.com/aquasecurity/tfsec) sur le `PATH`

### Build et tests

```bash
make test
make vet
make build
./bin/iac-sentinel version
```

### Scanner un plan

```bash
./bin/iac-sentinel scan -plan testdata/plan_mixed.json
./bin/iac-sentinel scan -plan plan.json -dir ./infra
./bin/iac-sentinel scan -plan plan.json -fail-on DANGER
```

| Flag | Signification |
|------|---------------|
| `-plan` | Chemin du JSON `terraform plan -json` (requis) |
| `-dir` | Dossier HCL pour Checkov/tfsec |
| `-fail-on` | `SAFE` / `CAUTION` / `DANGER` / `CRITICAL` — exit 1 si atteint |
| `-skip-checkov` / `-skip-tfsec` / `-skip-opa` | Desactiver un scanner |

### Exemples bibliotheque

```go
level := risk.Classify(tfplan.ActionDelete, "aws_db_instance")
// level == risk.CRITICAL
```

---

## Feuille de route

```mermaid
timeline
  title Chemin de livraison
  section Termine
    Parser Risk Policies : tfplan risk policy et Rego
  section En cours
    CLI : scan version renderer fail-on
  section Prevu
    Cout et ML : delta AWS et score logistique
    Explainer et livraison : LLM optionnel GitHub Action GoReleaser
```

1. Parseur de plan — fait
2. Classification de risque — fait
3. Moteur de policies (OPA + wrappers Checkov/tfsec) — fait
4. Rendu terminal + CLI `scan` / `version` + `--fail-on` — fait
5. Estimation statique du delta de cout AWS
6. Score de risque par regression logistique embarquee
7. Explainer LLM optionnel (`--no-ai`)
8. GitHub Action (commentaire PR) et distribution GoReleaser

---

## Notes de developpement

- Preferer les tests table-driven et les fixtures plutot que Terraform ou scanners live
- Wrapper les erreurs avec `%w` ; utiliser des erreurs sentinelle avec `errors.Is`
- Garder les packages sous `internal/` tant qu’une API publique n’est pas volontaire
- Les scanners optionnels doivent avertir, jamais planter, si le binaire est absent

---

## Licence

Apache 2.0 prevu (fichier de licence a ajouter).
