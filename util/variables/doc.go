// Package variables is the single source of truth for every variable that
// can appear in an Argo Workflows expression. It enforces correctness by
// construction: the type system makes it impossible to populate a variable
// without declaring it in the catalog.
//
// Core invariants
//
//  1. Every variable is declared exactly once, via Define, which returns the
//     handle (*Key) needed to write it.
//  2. A Scope's internal state is unexported. The only way to write a value
//     is through Key.Set; there is no exported map, no SetString, no write
//     helper that takes a bare string.
//  3. Registration is global: the catalog returned by All() is guaranteed to
//     contain every Key that any code path can possibly write, because no
//     code path can construct a Key by hand.
//
// Together these mean: the catalog and the write sites cannot drift —
// they are literally the same objects.
//
// Basic usage
//
//	// In util/variables/keys/globals.go (one declaration per variable):
//	var WorkflowName = variables.Define(variables.Spec{
//	    Template:    "workflow.name",
//	    Kind:        variables.KindGlobal,
//	    ValueType:   "string",
//	    AppliesTo:   []variables.TemplateKind{variables.TmplAll},
//	    Phases:      []variables.LifecyclePhase{variables.PhWorkflowStart},
//	    Description: "Workflow object name",
//	})
//
//	// At the write site:
//	scope := variables.NewScope()
//	keys.WorkflowName.Set(scope, wf.Name)
//
//	// At the read site:
//	name, _ := keys.WorkflowName.Get(scope)
//
// Parameterised keys
//
// Some variables are parameterised (steps.<name>.outputs.result). The
// template captures the placeholder name; Set/Get take the concrete value
// as a trailing positional arg:
//
//	var StepOutputResult = variables.Define(variables.Spec{
//	    Template: "steps.<name>.outputs.result",
//	    ...
//	})
//	StepOutputResult.Set(scope, "10", "generate")
//	v, _ := StepOutputResult.Get(scope, "generate")
//
// Interop with legacy map-consuming APIs
//
// template.Replace and similar take map[string]string or map[string]any.
// Scope exposes AsAnyMap and AsStringMap for read-only consumption at those
// boundaries. Writes never return through those maps — they are snapshots.
package variables
