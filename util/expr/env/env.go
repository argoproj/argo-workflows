package env

import (
	"encoding/json"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/evilmonkeyinc/jsonpath"
	"github.com/expr-lang/expr/builtin"

	"github.com/argoproj/argo-workflows/v3/util/expand"
)

// sprigFuncMap is a subset of Masterminds/sprig helpers.
// It keeps only those functions that do **not** have a direct equivalent
// in the Expr standard library and that we still consider safe to expose.
var sprigFuncMap map[string]interface{}

func init() {
	full := sprig.GenericFuncMap()

	allowed := map[string]struct{}{
		// Random / crypto
		"randAlpha":    {},
		"randAlphaNum": {},
		"randAscii":    {},
		"randNumeric":  {},
		"randBytes":    {},
		"randInt":      {},
		"uuidv4":       {},
		// Regex helpers
		"regexFindAll":           {},
		"regexSplit":             {},
		"regexReplaceAll":        {},
		"regexReplaceAllLiteral": {},
		"regexQuoteMeta":         {},
		// Text layout / case helpers
		"wrap":      {},
		"wrapWith":  {},
		"nospace":   {},
		"title":     {},
		"untitle":   {},
		"plural":    {},
		"initials":  {},
		"snakecase": {},
		"camelcase": {},
		"kebabcase": {},
		"swapcase":  {},
		"shuffle":   {},
		"trunc":     {},
		// Dict & reflection helpers
		"dict":           {},
		"set":            {},
		"deepCopy":       {},
		"merge":          {},
		"mergeOverwrite": {},
		"mergeRecursive": {},
		"dig":            {},
		"pluck":          {},
		"typeIsLike":     {},
		"kindIs":         {},
		"typeOf":         {},
		// Path / URL helpers
		"base":     {},
		"dir":      {},
		"ext":      {},
		"clean":    {},
		"urlParse": {},
		"urlJoin":  {},
		// SemVer helpers
		"semver":        {},
		"semverCompare": {},
		// Flow‑control helpers
		"fail":     {},
		"required": {},
		// Encoding / YAML helpers
		"b32enc":   {},
		"b32dec":   {},
		"toYaml":   {},
		"fromYaml": {},
	}

	// Build the curated func‑map
	sprigFuncMap = make(map[string]interface{}, len(allowed))
	for name, fn := range full {
		if _, ok := allowed[name]; ok {
			sprigFuncMap[name] = fn
		}
	}
}

func GetFuncMap(m map[string]interface{}) map[string]interface{} {
	env := expand.Expand(m)
	// Alias for the built-in `int` function, for backwards compatibility.
	env["asInt"] = builtin.Int
	// Alias for the built-in `float` function, for backwards compatibility.
	env["asFloat"] = builtin.Float
	env["jsonpath"] = jsonPath
	env["toJson"] = toJson
	env["sprig"] = sprigFuncMap
	return env
}

func toJson(v interface{}) string {
	output, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(output)
}

func jsonPath(jsonStr string, path string) interface{} {
	var jsonMap interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonMap)
	if err != nil {
		panic(err)
	}
	value, err := jsonpath.Query(path, jsonMap)
	if err != nil {
		panic(err)
	}
	return value
}
