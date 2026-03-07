package template

import (
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

// Replace takes a json-formatted string and performs variable replacement.
func Replace(s string, replaceMap map[string]string, allowUnresolved bool) (string, error) {
	if !json.Valid([]byte(s)) {
		return "", errors.New("cannot do template replacements with invalid JSON")
	}

	t, err := NewTemplate(s)
	if err != nil {
		return "", err
	}
	interReplaceMap := make(map[string]interface{})
	for k, v := range replaceMap {
		interReplaceMap[k] = v
	}
	replacedString, err := t.Replace(interReplaceMap, allowUnresolved)
	if err != nil {
		return s, err
	}

	if !json.Valid([]byte(replacedString)) {
		return s, errors.New("cannot finish template replacement because the result was invalid JSON")
	}

	return replacedString, nil
}

// ReplaceStrict behaves like Replace but enforces that tags starting with any of strictPrefixes MUST be resolved,
// even if allowUnresolved behavior is otherwise active (implicit).
func ReplaceStrict(s string, replaceMap map[string]string, strictPrefixes []string) (string, error) {
	if !json.Valid([]byte(s)) {
		return "", errors.New("cannot do template replacements with invalid JSON")
	}

	t, err := NewTemplate(s)
	if err != nil {
		return "", err
	}
	interReplaceMap := make(map[string]any)
	for k, v := range replaceMap {
		interReplaceMap[k] = v
	}
	replacedString, err := t.ReplaceStrict(interReplaceMap, strictPrefixes)
	if err != nil {
		return s, err
	}

	if !json.Valid([]byte(replacedString)) {
		return s, errors.New("cannot finish template replacement because the result was invalid JSON")
	}

	return replacedString, nil
}
