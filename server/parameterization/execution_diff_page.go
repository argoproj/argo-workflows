package parameterization

import (
	"fmt"
	"maps"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// Page only after executionDiff has reauthorized and validated the complete
// source and target. A status-only resourceVersion change must not invalidate
// pages, but changing either input identity or a live definition must do so.
func executionDiffPage(full *ExecutionComparison, req ExecutionDiffRequest) (*ExecutionComparison, error) {
	identity := *full
	identity.ResourceVersion = ""
	snapshot := p.Digest(identity)
	if req.Snapshot != "" && req.Snapshot != snapshot {
		return nil, fmt.Errorf("execution comparison changed; restart comparison")
	}
	parameters := full.Parameters.(map[string]any)
	keys := []string{"changes", "modelChanges", "sourceChanges"}
	sections := make([][]p.Change, 0, len(keys)+len(full.Layers))
	totals := map[string]int{}
	for _, key := range keys {
		items := parameters[key].([]p.Change)
		sections = append(sections, items)
		totals[key] = len(items)
	}
	for _, layer := range full.Layers {
		sections = append(sections, layer.Changes)
	}
	pages, pagination, err := changePages(sections, snapshot, req.Limit, req.Continue)
	if err != nil {
		return nil, err
	}
	out := *full
	parameterPage := map[string]any{}
	maps.Copy(parameterPage, parameters)
	for i, key := range keys {
		parameterPage[key] = pages[i]
	}
	parameterPage["totals"] = totals
	out.Parameters = parameterPage
	out.Layers = append([]ExecutionDiffLayer(nil), full.Layers...)
	for i := range out.Layers {
		out.Layers[i].Total = len(full.Layers[i].Changes)
		out.Layers[i].Changes = pages[len(keys)+i]
	}
	out.Pagination = pagination
	return &out, nil
}
