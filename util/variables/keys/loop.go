package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// item / item.<key> — loop iteration values (withItems/withParam).
func item(template, valueType, description string) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindItem, ValueType: valueType, AppliesTo: anyTmpl,
		Phases:      []v.LifecyclePhase{v.PhInsideLoop, v.PhDuringExecute},
		Description: description,
	})
}

var (
	Item      = item("item", "string|json", "Current loop iteration value (withItems/withParam). JSON for map/list items.")
	ItemByKey = item("item.<key>", "string", "Accessor into a map-typed loop iteration value")
)
