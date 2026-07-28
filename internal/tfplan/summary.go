package tfplan

import "slices"

// Summary counts mutating changes and lists them sorted by address.
type Summary struct {
	Creates  int
	Updates  int
	Replaces int
	Deletes  int
	Changes  []ResourceChange
}

// Summarize counts create/update/replace/delete.
// No-ops, reads, and data sources are excluded. A nil plan yields zero.
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
