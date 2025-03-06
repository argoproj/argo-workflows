package util

import (
	"strings"
)

// GenerateBackfillWorkflowPrefix return a backfill workflow prefix
func GenerateBackfillWorkflowPrefix(cronWorkflowName, ops string) string {
	prefix := cronWorkflowName + "-backfill-" + strings.ToLower(ops)
	prefix = ensureWorkflowNamePrefixLength(prefix)
	return prefix
}

func ensureWorkflowNamePrefixLength(prefix string) string {
	if len(prefix) > maxPrefixLength-1 {
		return prefix[0 : maxPrefixLength-1]
	}

	return prefix
}
