package tfplan

import "slices"

// Summary aggregates mutating changes from a plan.
type Summary struct {
	Creates  int
	Updates  int
	Replaces int
	Deletes  int
	Changes  []ResourceChange // sorted by Address for stable output
}

// Summarize counts create/update/replace/delete and returns them sorted.
// No-ops, reads, and data sources are skipped. A nil plan yields zero.
func Summarize(p *Plan) Summary {
	if p == nil {
		return Summary{}
	}

	var s Summary
	for _, rc := range p.ResourceChanges {
		if rc.Mode == "data" {
			continue
		}
		switch rc.Change.Action() {
		case ActionCreate:
			s.Creates++
			s.Changes = append(s.Changes, rc)
		case ActionUpdate:
			s.Updates++
			s.Changes = append(s.Changes, rc)
		case ActionReplace:
			s.Replaces++
			s.Changes = append(s.Changes, rc)
		case ActionDelete:
			s.Deletes++
			s.Changes = append(s.Changes, rc)
		}
	}

	slices.SortFunc(s.Changes, func(a, b ResourceChange) int {
		switch {
		case a.Address < b.Address:
			return -1
		case a.Address > b.Address:
			return 1
		default:
			return 0
		}
	})
	return s
}
