package template

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

func IsMissingVariableErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "failed to resolve {{") {
		return true
	}
	if strings.Contains(msg, " is missing") {
		return true
	}
	if strings.Contains(msg, "variable not in env") {
		return true
	}
	return false
}

// ToAnyMap widens a string map for use with Replace. Callers that track absent optionals
// (skipped/omitted node outputs) build a map[string]any directly so nil entries survive.
func ToAnyMap(m map[string]string) map[string]any {
	anyMap := make(map[string]any, len(m))
	for k, v := range m {
		anyMap[k] = v
	}
	return anyMap
}

// Replace takes a json-formatted string and performs variable replacement. Values are raw: a nil
// entry marks an absent optional (a skipped/omitted node's output with no default), and a simple
// tag resolving to nil is a terminal error regardless of allowUnresolved, which only governs tags
// whose key is missing entirely.
func Replace(ctx context.Context, s string, replaceMap map[string]any, allowUnresolved bool) (string, error) {
	if !json.Valid([]byte(s)) {
		return "", errors.New("cannot do template replacements with invalid JSON")
	}

	t, err := NewTemplate(s)
	if err != nil {
		return "", err
	}
	replacedString, err := t.Replace(ctx, replaceMap, allowUnresolved)
	if err != nil {
		return s, err
	}

	if !json.Valid([]byte(replacedString)) {
		return s, errors.New("cannot finish template replacement because the result was invalid JSON")
	}

	return replacedString, nil
}

// ReplaceStrictAny behaves like Replace but enforces that tags starting with any of strictPrefixes
// MUST be resolved, even if allowUnresolved behavior is otherwise active (implicit). It takes raw
// values, preserving nil entries so that expression tags can distinguish an absent (nil) value from
// an empty string (e.g. via `??`). A simple tag resolving to a nil value is a terminal error (not a
// missing-variable error): an absent optional must be handled by a producer default, a consumer
// input default (dropping the argument before substitution), or an expression fallback.
func ReplaceStrictAny(ctx context.Context, s string, replaceMap map[string]any, strictPrefixes []string) (string, error) {
	if !json.Valid([]byte(s)) {
		return "", errors.New("cannot do template replacements with invalid JSON")
	}

	t, err := NewTemplate(s)
	if err != nil {
		return "", err
	}
	replacedString, err := t.ReplaceStrict(ctx, replaceMap, strictPrefixes)
	if err != nil {
		return s, err
	}

	if !json.Valid([]byte(replacedString)) {
		return s, errors.New("cannot finish template replacement because the result was invalid JSON")
	}

	return replacedString, nil
}
