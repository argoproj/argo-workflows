package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// item / item.<key> — loop iteration values (withItems/withParam).
// Bound only inside a withItems/withParam expansion; the during-execute
// phase is implied (loop expansion is a sub-phase of template execution),
// so it isn't listed separately — that would falsely surface item under
// the generic "during-execute" phase grouping in the catalog and let the
// matrix mark it reachable from contexts (e.g. the exit handler) that
// never run under withItems.
func item(template, valueType, description string) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindItem, ValueType: valueType, AppliesTo: anyTmpl,
		Phases:      []v.LifecyclePhase{v.PhInsideLoop},
		Description: description,
	})
}

var (
	Item      = item("item", "string or json", "Current loop iteration value (withItems/withParam). JSON for map/list items.")
	ItemByKey = item("item.<key>", "string", "Accessor into a map-typed loop iteration value")
)
