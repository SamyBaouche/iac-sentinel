package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SamyBaouche/iac-sentinel/internal/tfplan"
	"github.com/SamyBaouche/iac-sentinel/policies"
	"github.com/open-policy-agent/opa/v1/rego"
)

// EvaluateOPA runs the embedded Rego policies against input.
//
// input is usually the map produced by PlanInput (resource_changes, …).
// OPA evaluates every package under data.sentinel.*.violation.
func EvaluateOPA(ctx context.Context, input any) (Result, error) {
	modules, err := loadEmbeddedModules()
	if err != nil {
		return Result{}, err
	}

	opts := []func(*rego.Rego){
		rego.Query("data.sentinel[pkg].violation[v]"),
		rego.Input(input),
	}
	opts = append(opts, modules...)

	rs, err := rego.New(opts...).Eval(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("policy: opa eval: %w", err)
	}

	var findings []Finding
	for _, result := range rs {
		// Query "data.sentinel[pkg].violation[v]" binds each hit to "v".
		raw, ok := result.Bindings["v"]
		if !ok {
			continue
		}
		f, err := findingFromOPA(raw)
		if err != nil {
			continue
		}
		findings = append(findings, f)
	}

	if len(findings) == 0 {
		findings = findingsFromExpressions(rs)
	}

	return Result{Findings: findings}, nil
}

func loadEmbeddedModules() ([]func(*rego.Rego), error) {
	entries, err := policies.FS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("policy: read embedded policies: %w", err)
	}

	var mods []func(*rego.Rego)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".rego") {
			continue
		}
		body, err := policies.FS.ReadFile(e.Name())
		if err != nil {
			return nil, fmt.Errorf("policy: read %s: %w", e.Name(), err)
		}
		name := e.Name()
		mods = append(mods, rego.Module(name, string(body)))
	}
	if len(mods) == 0 {
		return nil, fmt.Errorf("policy: no .rego modules embedded")
	}
	return mods, nil
}

func findingsFromExpressions(rs rego.ResultSet) []Finding {
	var out []Finding
	for _, result := range rs {
		for _, expr := range result.Expressions {
			switch v := expr.Value.(type) {
			case []any:
				for _, item := range v {
					if f, err := findingFromOPA(item); err == nil {
						out = append(out, f)
					}
				}
			case map[string]any:
				if f, err := findingFromOPA(v); err == nil {
					out = append(out, f)
				}
			}
		}
	}
	return out
}

func findingFromOPA(raw any) (Finding, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return Finding{}, fmt.Errorf("not an object")
	}
	return Finding{
		ID:          asString(m["id"]),
		Source:      SourceOPA,
		Severity:    normalizeSeverity(asString(m["severity"])),
		Title:       asString(m["title"]),
		Description: asString(m["description"]),
		Resource:    asString(m["resource"]),
	}, nil
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

// PlanInput converts a parsed Terraform plan into a generic map for OPA.
// json.RawMessage fields (before/after) become real JSON values.
func PlanInput(p *tfplan.Plan) (map[string]any, error) {
	if p == nil {
		return map[string]any{"resource_changes": []any{}}, nil
	}

	changes := make([]any, 0, len(p.ResourceChanges))
	for _, rc := range p.ResourceChanges {
		after, err := rawToAny(rc.Change.After)
		if err != nil {
			return nil, fmt.Errorf("policy: decode after for %s: %w", rc.Address, err)
		}
		before, err := rawToAny(rc.Change.Before)
		if err != nil {
			return nil, fmt.Errorf("policy: decode before for %s: %w", rc.Address, err)
		}
		changes = append(changes, map[string]any{
			"address":       rc.Address,
			"mode":          rc.Mode,
			"type":          rc.Type,
			"name":          rc.Name,
			"provider_name": rc.ProviderName,
			"change": map[string]any{
				"actions": rc.Change.Actions,
				"before":  before,
				"after":   after,
			},
		})
	}

	return map[string]any{
		"format_version":    p.FormatVersion,
		"terraform_version": p.TerraformVersion,
		"resource_changes":  changes,
	}, nil
}

func rawToAny(raw json.RawMessage) (any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return v, nil
}
