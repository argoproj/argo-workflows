package parameterization

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

type DiffRequest struct {
	Left     string `json:"left"`
	Right    string `json:"right"`
	Limit    int    `json:"limit,omitempty"`
	Continue string `json:"continue,omitempty"`
}

type diffCursor struct {
	Snapshot string `json:"snapshot"`
	Offset   int    `json:"offset"`
}

// Pagination follows validation and caller-scoped reads on every request. The
// cursor is a position, never an authorization grant or a cached document.
func diffPage(comparison map[string]any, request DiffRequest) (map[string]any, error) {
	keys := []string{"changes", "modelChanges", "sourceChanges"}
	sections := make([][]p.Change, len(keys))
	totals := map[string]int{}
	for i, key := range keys {
		sections[i] = comparison[key].([]p.Change)
		totals[key] = len(sections[i])
	}
	pages, pagination, err := changePages(sections, p.Digest(comparison), request.Limit, request.Continue)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	maps.Copy(out, comparison)
	for i, key := range keys {
		out[key] = pages[i]
	}
	out["totals"] = totals
	out["pagination"] = pagination
	return out, nil
}

func changePages(sections [][]p.Change, snapshot string, limit int, token string) ([][]p.Change, map[string]any, error) {
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 200 {
		return nil, nil, fmt.Errorf("diff limit must be between 1 and 200")
	}
	total := 0
	for _, section := range sections {
		total += len(section)
	}
	offset := 0
	if token != "" {
		if len(token) > 512 {
			return nil, nil, fmt.Errorf("invalid diff cursor")
		}
		raw, err := base64.RawURLEncoding.DecodeString(token)
		var cursor diffCursor
		if err != nil || json.Unmarshal(raw, &cursor) != nil || cursor.Snapshot != snapshot || cursor.Offset < 0 || cursor.Offset >= total {
			return nil, nil, fmt.Errorf("diff cursor does not match this comparison; restart comparison")
		}
		offset = cursor.Offset
	}
	out := make([][]p.Change, len(sections))
	remaining, position := limit, 0
	for i, items := range sections {
		start := min(max(offset-position, 0), len(items))
		count := min(len(items)-start, remaining)
		out[i] = items[start : start+count]
		remaining -= count
		position += len(items)
	}
	next := ""
	end := offset + limit - remaining
	if end < total {
		raw, _ := json.Marshal(diffCursor{Snapshot: snapshot, Offset: end})
		next = base64.RawURLEncoding.EncodeToString(raw)
	}
	return out, map[string]any{"total": total, "offset": offset, "limit": limit, "continue": next, "snapshot": snapshot}, nil
}
