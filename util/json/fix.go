package json

import "strings"

func Fix(s string) string {
	// https://stackoverflow.com/questions/28595664/how-to-stop-json-marshal-from-escaping-and/28596225
	s = strings.Replace(s, "\\u003c", "<", -1)
	s = strings.Replace(s, "\\u003e", ">", -1)
	s = strings.Replace(s, "\\u0026", "&", -1)
	return s
}
