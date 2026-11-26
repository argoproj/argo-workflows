package lint

import (
	"encoding/json"
	"fmt"
)

// JsonLintResult is the intermediate struct to convert result to json
type JsonLintResult struct {
	File    string   `json:"file"`
	ErrsStr []string `json:"errors"`
	Linted  bool     `json:"linted"`
}

// JsonLintResults is the intermediate struct to convert results to json
type JsonLintResults struct {
	Results        []*JsonLintResult `json:"results"`
	Success        bool              `json:"success"`
	AnythingLinted bool              `json:"anything_linted"`
}

type formatterJson struct{}

func (f formatterJson) Format(l *LintResult) string {
	return ""
}

func (f formatterJson) Summarize(l *LintResults) string {
	b, err := json.Marshal(toJsonResultStruct(l))
	if err != nil {
		return fmt.Sprintf("Failed to marshal results to JSON: %e", err)
	}
	return string(b)
}

func toJsonResultStruct(l *LintResults) *JsonLintResults {
	jsonLintResults := &JsonLintResults{
		Results:        make([]*JsonLintResult, len(l.Results)),
		Success:        l.Success,
		AnythingLinted: l.anythingLinted,
	}

	for i, lr := range l.Results {
		errStrs := make([]string, len(lr.Errs))
		for j, err := range lr.Errs {
			errStrs[j] = err.Error()
		}
		jsonLintResults.Results[i] = &JsonLintResult{
			File:    lr.File,
			Linted:  lr.Linted,
			ErrsStr: errStrs,
		}
	}

	return jsonLintResults
}
