// Package variables is the catalog of every Argo Workflows expression
// variable. Each variable is declared once via Define (in
// util/variables/keys/) and obtains a *Key handle. Key.Set is the only
// public write path on a *Scope, so the catalog and the write sites cannot
// drift — they are literally the same objects.
package variables
