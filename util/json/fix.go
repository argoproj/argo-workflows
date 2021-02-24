package json

import "strings"

func Fix(s string) string {
	// I've never encountered something so fundamentally broken in any programming language as the spacing of HTML characters in Golang's JSON parsing.
	// Any code that utilize will JSON in Golang will always have edge-case bugs related to >, < and &
	s = strings.Replace(s, "\\u003c", "<", -1)
	s = strings.Replace(s, "\\u003e", ">", -1)
	s = strings.Replace(s, "\\u0026", "&", -1)
	return s
}
