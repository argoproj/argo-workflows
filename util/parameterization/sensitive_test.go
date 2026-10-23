package parameterization

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func sensitiveArrayModel() Model {
	return Model{Dialect: Dialect, Schema: map[string]any{"type": "object", "properties": map[string]any{"items": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}, "credential": map[string]any{"type": "object"}}}}}}, SensitivePaths: []string{"/items/*/credential"}, IdentityRules: []IdentityRule{{Path: "/items", KeyPath: "/id"}}}
}
func TestSensitiveArraysValidateEveryRowAndReferenceEncoding(t *testing.T) {
	m := sensitiveArrayModel()
	ref := map[string]any{"secretRef": map[string]any{"name": "synthetic", "key": "value"}}
	values := map[string]any{"items": []any{map[string]any{"id": "one", "credential": ref}, map[string]any{"id": "two", "credential": ref}}}
	_, _, errors := Resolve(m, values)
	require.Empty(t, errors)
	for _, path := range []string{"", "/items", "/items/0", "/items/1/credential", "/items/1/credential/secretRef/name"} {
		_, err := Arguments(m, values, []Mapping{{Path: path, Parameter: "items", Encoding: "json"}})
		require.Error(t, err, path)
	}
	args, err := Arguments(m, values, []Mapping{{Path: "/items", Parameter: "items", Encoding: "secret-reference-json"}})
	require.NoError(t, err)
	require.Contains(t, args["items"], "secretRef")
	_, err = Arguments(m, values, []Mapping{{Path: "/items/0/credential/secretRef/name", Parameter: "items", Encoding: "secret-reference-json"}})
	require.Error(t, err)
	for _, bad := range []any{"private-value", map[string]any{"secretRef": ref["secretRef"], "value": "private-value"}} {
		values["items"].([]any)[1].(map[string]any)["credential"] = bad
		_, _, errors = Resolve(m, values)
		require.Equal(t, "SecretReferenceRequired", errors[0].Code)
		require.Equal(t, "/items/1/credential", errors[0].Path)
	}
	for _, malformed := range []any{"private-value", map[string]any{"0": map[string]any{"credential": "private-value"}}, []any{"private-value"}} {
		values["items"] = malformed
		_, _, errors = Resolve(m, values)
		require.Equal(t, "SecretReferenceRequired", errors[0].Code)
	}
	m.IdentityRules = nil
	require.NotEmpty(t, ValidateModelSchema(m))
}
func TestSensitiveNestedArraysAndEscapedPointers(t *testing.T) {
	v := map[string]any{"a/b": []any{map[string]any{"rows": []any{map[string]any{"~token": "private-value"}}}}}
	e := ValidateSensitiveReferences([]string{"/a~1b/*/rows/*/~0token"}, v)
	require.Len(t, e, 1)
	require.Equal(t, "/a~1b/0/rows/0/~0token", e[0].Path)
	require.False(t, MatchPath("/a/*/token", "/a/01/token"))
	require.False(t, MatchPath("/a/*/token", "/a/name/token"))
	require.True(t, PathsOverlap("/a/*/token", "/a/12/token/secretRef"))
	require.False(t, PathsOverlap("/a/*/token", "/a/1/public"))
	m := sensitiveArrayModel()
	m.Derived = []Derivation{{Path: "/copy", From: "/items"}}
	require.NotEmpty(t, ValidateModelSchema(m))
}

func TestLatestSecretReferenceIsExplicitAndReferenceOnly(t *testing.T) {
	good := map[string]any{"credential": map[string]any{"latestSecretRef": map[string]any{"configMap": "versions", "key": "current"}}}
	require.Empty(t, ValidateSensitiveReferences([]string{"/credential"}, good))
	for _, bad := range []any{
		map[string]any{"latestSecretRef": map[string]any{"configMap": "versions", "key": "current", "namespace": "other"}},
		map[string]any{"latestSecretRef": map[string]any{"configMap": "", "key": "current"}},
		map[string]any{"latestSecretRef": map[string]any{"name": "wrong-shape", "key": "current"}},
		map[string]any{"latestSecretRef": map[string]any{"configMap": "versions", "key": "current"}, "value": "private"},
	} {
		require.NotEmpty(t, ValidateSensitiveReferences([]string{"/credential"}, map[string]any{"credential": bad}))
	}
}
