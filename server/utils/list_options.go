package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type ListOptions struct {
	Namespace, Name              string
	NamePrefix, NameFilter       string
	NamespaceFilter              string
	MinStartedAt, MaxStartedAt   time.Time
	CreatedAfter, FinishedBefore time.Time
	LabelRequirements            labels.Requirements
	Limit, Offset                int
	ShowRemainingItemCount       bool
	StartedAtAscending           bool
	// CursorStartedAt and CursorUID enable keyset pagination for archived
	// workflows. When set, the query uses WHERE (startedat, uid) < (cursor)
	// instead of OFFSET, providing constant-time pagination.
	CursorStartedAt time.Time
	CursorUID       string
}

func (l ListOptions) WithLimit(limit int) ListOptions {
	l.Limit = limit
	return l
}

func (l ListOptions) WithOffset(offset int) ListOptions {
	l.Offset = offset
	return l
}

func (l ListOptions) WithShowRemainingItemCount(showRemainingItemCount bool) ListOptions {
	l.ShowRemainingItemCount = showRemainingItemCount
	return l
}

func (l ListOptions) WithMaxStartedAt(maxStartedAt time.Time) ListOptions {
	l.MaxStartedAt = maxStartedAt
	return l
}

func (l ListOptions) WithMinStartedAt(minStartedAt time.Time) ListOptions {
	l.MinStartedAt = minStartedAt
	return l
}

func (l ListOptions) WithStartedAtAscending(ascending bool) ListOptions {
	l.StartedAtAscending = ascending
	return l
}

// archivedWorkflowCursor represents a keyset pagination cursor for archived
// workflows. It encodes the position using startedat and uid for deterministic
// ordering.
type archivedWorkflowCursor struct {
	StartedAt time.Time `json:"startedat"`
	UID       string    `json:"uid"`
}

// EncodeArchivedWorkflowCursor creates a base64-encoded cursor token from a
// startedat timestamp and uid.
func EncodeArchivedWorkflowCursor(startedAt time.Time, uid string) string {
	cursor := archivedWorkflowCursor{StartedAt: startedAt, UID: uid}
	data, _ := json.Marshal(cursor)
	return "c:" + base64.StdEncoding.EncodeToString(data)
}

// DecodeArchivedWorkflowCursor attempts to decode a continue token as an
// archived workflow cursor. Returns the cursor and true if successful, or
// zero values and false if the token is not a cursor.
func DecodeArchivedWorkflowCursor(continueToken string) (archivedWorkflowCursor, bool) {
	if !strings.HasPrefix(continueToken, "c:") {
		return archivedWorkflowCursor{}, false
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(continueToken, "c:"))
	if err != nil {
		return archivedWorkflowCursor{}, false
	}
	var cursor archivedWorkflowCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return archivedWorkflowCursor{}, false
	}
	return cursor, true
}

func BuildListOptions(options metav1.ListOptions, ns, namePrefix, nameFilter, createdAfter, finishedBefore string) (ListOptions, error) {
	if options.Continue == "" {
		options.Continue = "0"
	}

	limit := int(options.Limit)

	// Try to decode as an archived workflow cursor first (keyset pagination).
	// Fall back to offset-based parsing for backward compatibility.
	var offset int
	var cursorStartedAt time.Time
	var cursorUID string

	if cursor, ok := DecodeArchivedWorkflowCursor(options.Continue); ok {
		cursorStartedAt = cursor.StartedAt
		cursorUID = cursor.UID
		offset = 0 // offset is not used with keyset pagination
	} else {
		var err error
		offset, err = strconv.Atoi(options.Continue)
		if err != nil {
			// no need to use sutils here
			return ListOptions{}, status.Error(codes.InvalidArgument, "listOptions.continue must be int or cursor")
		}
		if offset < 0 {
			// no need to use sutils here
			return ListOptions{}, status.Error(codes.InvalidArgument, "listOptions.continue must >= 0")
		}
	}

	// namespace is now specified as its own query parameter
	// note that for backward compatibility, the field selector 'metadata.namespace' is also supported for now
	namespace := ns // optional
	namespaceFilter := ""
	name := ""
	minStartedAt := time.Time{}
	maxStartedAt := time.Time{}
	createdAfterTime := time.Time{}
	finishedBeforeTime := time.Time{}
	var err error

	if createdAfter != "" {
		createdAfterTime, err = time.Parse(time.RFC3339, createdAfter)
		if err != nil {
			return ListOptions{}, ToStatusError(err, codes.Internal)
		}
	}
	if finishedBefore != "" {
		finishedBeforeTime, err = time.Parse(time.RFC3339, finishedBefore)
		if err != nil {
			return ListOptions{}, ToStatusError(err, codes.Internal)
		}
	}
	showRemainingItemCount := false
	for selector := range strings.SplitSeq(options.FieldSelector, ",") {
		if len(selector) == 0 {
			continue
		}
		if after, ok := strings.CutPrefix(selector, "metadata.namespace!="); ok {
			namespace = after
			namespaceFilter = "NotEquals"
		} else if after, ok := strings.CutPrefix(selector, "metadata.namespace=="); ok {
			fieldSelectedNamespace := after
			switch namespace {
			case "":
				namespace = fieldSelectedNamespace
			case fieldSelectedNamespace:
				// namespace matches, nothing to do
			default:
				return ListOptions{}, status.Errorf(codes.InvalidArgument,
					"'namespace' query param (%q) and fieldselector 'metadata.namespace' (%q) are both specified and contradict each other", namespace, fieldSelectedNamespace)
			}
		} else if after, ok := strings.CutPrefix(selector, "metadata.namespace="); ok {
			// for backward compatibility, the field selector 'metadata.namespace' is supported for now despite the addition
			// of the new 'namespace' query parameter, which is what the UI uses
			fieldSelectedNamespace := after
			switch namespace {
			case "":
				namespace = fieldSelectedNamespace
			case fieldSelectedNamespace:
				// namespace matches, nothing to do
			default:
				return ListOptions{}, status.Errorf(codes.InvalidArgument,
					"'namespace' query param (%q) and fieldselector 'metadata.namespace' (%q) are both specified and contradict each other", namespace, fieldSelectedNamespace)
			}
		} else if after, ok := strings.CutPrefix(selector, "metadata.name!="); ok {
			name = after
			nameFilter = "NotEquals"
		} else if after, ok := strings.CutPrefix(selector, "metadata.name=="); ok {
			name = after
		} else if after, ok := strings.CutPrefix(selector, "metadata.name="); ok {
			name = after
		} else if after, ok := strings.CutPrefix(selector, "spec.startedAt>"); ok {
			minStartedAt, err = time.Parse(time.RFC3339, after)
			if err != nil {
				// startedAt is populated by us, it should therefore be valid.
				return ListOptions{}, ToStatusError(err, codes.Internal)
			}
		} else if after, ok := strings.CutPrefix(selector, "spec.startedAt<"); ok {
			maxStartedAt, err = time.Parse(time.RFC3339, after)
			if err != nil {
				// no need to use sutils here
				return ListOptions{}, ToStatusError(err, codes.Internal)
			}
		} else if strings.HasPrefix(selector, "ext.showRemainingItemCount") {
			showRemainingItemCount, err = strconv.ParseBool(strings.TrimPrefix(selector, "ext.showRemainingItemCount="))
			if err != nil {
				// populated by us, it should therefore be valid.
				return ListOptions{}, ToStatusError(err, codes.Internal)
			}
		} else {
			return ListOptions{}, ToStatusError(fmt.Errorf("unsupported requirement %s", selector), codes.InvalidArgument)
		}
	}
	requirements, err := labels.ParseToRequirements(options.LabelSelector)
	if err != nil {
		return ListOptions{}, ToStatusError(err, codes.InvalidArgument)
	}
	return ListOptions{
		Namespace:              namespace,
		Name:                   name,
		NamePrefix:             namePrefix,
		NameFilter:             nameFilter,
		NamespaceFilter:        namespaceFilter,
		CreatedAfter:           createdAfterTime,
		FinishedBefore:         finishedBeforeTime,
		MinStartedAt:           minStartedAt,
		MaxStartedAt:           maxStartedAt,
		LabelRequirements:      requirements,
		Limit:                  limit,
		Offset:                 offset,
		ShowRemainingItemCount: showRemainingItemCount,
		CursorStartedAt:        cursorStartedAt,
		CursorUID:              cursorUID,
	}, nil
}
