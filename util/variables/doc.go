// Package variables is the catalog of every Argo Workflows expression
// variable. Each variable is declared once via Define (in
// util/variables/keys/) and obtains a *Key handle. A *Scope's storage is
// unexported, so Key.Set and Key.SetSkipped are its only write paths and
// both key entries through the catalog — the Scope write sites therefore
// cannot drift from the catalog.
//
// Scopes that are not a *Scope (the metric, loop and retry substitution
// maps are plain map[string]string) still derive their keys from the
// catalog via Key.Template / Key.Concretize, but, being ordinary maps, are
// not structurally constrained to it.
package variables
