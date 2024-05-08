package utils

import (
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
	Namespace, Name, NamePrefix string
	MinStartedAt, MaxStartedAt  time.Time
	LabelRequirements           labels.Requirements
	Limit, Offset               int
	ShowRemainingItemCount      bool
	StartedAtAscending          bool
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

func BuildListOptions(options metav1.ListOptions, ns, namePrefix string) (ListOptions, error) {
	if options.Continue == "" {
		options.Continue = "0"
	}
	limit := int(options.Limit)

	offset, err := strconv.Atoi(options.Continue)
	if err != nil {
		// no need to use sutils here
		return ListOptions{}, status.Error(codes.InvalidArgument, "listOptions.continue must be int")
	}
	if offset < 0 {
		// no need to use sutils here
		return ListOptions{}, status.Error(codes.InvalidArgument, "listOptions.continue must >= 0")
	}

	// namespace is now specified as its own query parameter
	// note that for backward compatibility, the field selector 'metadata.namespace' is also supported for now
	namespace := ns // optional
	name := ""
	minStartedAt := time.Time{}
	maxStartedAt := time.Time{}
	showRemainingItemCount := false
	for _, selector := range strings.Split(options.FieldSelector, ",") {
		if len(selector) == 0 {
			continue
		}
		if strings.HasPrefix(selector, "metadata.namespace=") {
			// for backward compatibility, the field selector 'metadata.namespace' is supported for now despite the addition
			// of the new 'namespace' query parameter, which is what the UI uses
			fieldSelectedNamespace := strings.TrimPrefix(selector, "metadata.namespace=")
			switch namespace {
			case "":
				namespace = fieldSelectedNamespace
			case fieldSelectedNamespace:
				break
			default:
				return ListOptions{}, status.Errorf(codes.InvalidArgument,
					"'namespace' query param (%q) and fieldselector 'metadata.namespace' (%q) are both specified and contradict each other", namespace, fieldSelectedNamespace)
			}
		} else if strings.HasPrefix(selector, "metadata.name=") {
			name = strings.TrimPrefix(selector, "metadata.name=")
		} else if strings.HasPrefix(selector, "spec.startedAt>") {
			minStartedAt, err = time.Parse(time.RFC3339, strings.TrimPrefix(selector, "spec.startedAt>"))
			if err != nil {
				// startedAt is populated by us, it should therefore be valid.
				return ListOptions{}, ToStatusError(err, codes.Internal)
			}
		} else if strings.HasPrefix(selector, "spec.startedAt<") {
			maxStartedAt, err = time.Parse(time.RFC3339, strings.TrimPrefix(selector, "spec.startedAt<"))
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
		MinStartedAt:           minStartedAt,
		MaxStartedAt:           maxStartedAt,
		LabelRequirements:      requirements,
		Limit:                  limit,
		Offset:                 offset,
		ShowRemainingItemCount: showRemainingItemCount,
	}, nil
}
