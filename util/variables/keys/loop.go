package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// ─────────────────────────── item / item.<key> — loop iteration values

var (
	Item = v.Define(v.Spec{
		Template:    "item",
		Kind:        v.KindItem,
		ValueType:   "string|json",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhInsideLoop, v.PhDuringExecute},
		Description: "Current loop iteration value (withItems/withParam). JSON for map/list items.",
	})
	ItemByKey = v.Define(v.Spec{
		Template:    "item.<key>",
		Kind:        v.KindItem,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhInsideLoop, v.PhDuringExecute},
		Description: "Accessor into a map-typed loop iteration value",
	})
)
