package template

import (
	"encoding/json"
	"errors"
)

// Replace takes a json-formatted string and performs variable replacement.
func Replace(s string, replaceMap map[string]string, allowUnresolved bool) (string, error) {
	if !json.Valid([]byte(s)) {
		return "", errors.New("cannot do template replacements with invalid JSON")
	}

	t, err := NewTemplate(s)
	if err != nil {
		return "", err
	}

	replacedString, err := t.Replace(replaceMap, allowUnresolved)
	if err != nil {
		return s, err
	}

	if !json.Valid([]byte(replacedString)) {
		return s, errors.New("cannot finish template replacement because the result was invalid JSON")
	}

	return replacedString, nil
}
