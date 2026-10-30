package parameterization

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleModel() Model {
	return Model{Dialect: Dialect, Schema: map[string]any{"type": "object", "properties": map[string]any{"enabled": map[string]any{"type": "boolean", "default": true}, "count": map[string]any{"type": "integer", "default": 4}, "nullable": map[string]any{"type": []any{"string", "null"}, "default": "fallback"}, "text": map[string]any{"type": "string", "default": "fallback"}, "tags.example/key": map[string]any{"type": "string"}}, "additionalProperties": true}}
}
func TestResolveExplicitValuesAndEscapedPaths(t *testing.T) {
	m := sampleModel()
	input := map[string]any{"enabled": false, "count": 0, "nullable": nil, "text": "", "tags.example/key": true}
	value, sources, errs := Resolve(m, input)
	v := value.(map[string]any)
	require.Equal(t, false, v["enabled"])
	require.InDelta(t, float64(0), v["count"], 0)
	require.Nil(t, v["nullable"])
	require.Empty(t, v["text"])
	require.Equal(t, "user", sources["/enabled"])
	require.Equal(t, "user", sources["/nullable"])
	require.Len(t, errs, 1)
	require.Equal(t, "/tags.example~1key", errs[0].Path)
	require.Equal(t, 0, input["count"], "resolution must not modify caller's input")
}
func TestSchemaReferencesDefaultsAndCycle(t *testing.T) {
	m := Model{Dialect: Dialect, Schema: map[string]any{"type": "object", "properties": map[string]any{"field": map[string]any{"$ref": "#/definitions/defaulted"}}, "definitions": map[string]any{"defaulted": map[string]any{"type": "string", "default": "yes"}}}}
	v, src, errs := Resolve(m, map[string]any{})
	require.Empty(t, errs)
	require.Equal(t, "yes", v.(map[string]any)["field"])
	require.Equal(t, "default", src["/field"])
	m.Schema["definitions"].(map[string]any)["defaulted"] = map[string]any{"$ref": "#/definitions/defaulted"}
	_, _, errs = Resolve(m, map[string]any{})
	require.Equal(t, "InvalidSchema", errs[0].Code)
	m.Schema["properties"].(map[string]any)["field"] = map[string]any{"$ref": "https://example.invalid/schema"}
	_, _, errs = Resolve(m, nil)
	require.Equal(t, "UnsupportedReference", errs[0].Code)
}
func TestSecretsCannotEnterArgumentsOrOrdinaryRevisions(t *testing.T) {
	m := sampleModel()
	m.SensitivePaths = []string{"/credentials"}
	for _, value := range []any{"plaintext", map[string]any{"secretRef": map[string]any{"name": "secret", "key": "token"}, "value": "plaintext"}, map[string]any{"secretRef": map[string]any{"name": "secret", "key": "token", "value": "plaintext"}}} {
		_, _, errs := Resolve(m, map[string]any{"credentials": value})
		require.Equal(t, "SecretReferenceRequired", errs[len(errs)-1].Code)
	}
	value, _, errs := Resolve(m, map[string]any{"credentials": map[string]any{"secretRef": map[string]any{"name": "secret", "key": "token"}}})
	require.Empty(t, errs)
	for _, path := range []string{"", "/credentials", "/credentials/secretRef/name"} {
		_, err := Arguments(m, value, []Mapping{{Path: path, Parameter: "unsafe", Encoding: "json"}})
		require.Error(t, err)
	}
}
func TestSemanticDiffCollectionIdentityAndNull(t *testing.T) {
	before := map[string]any{"items": []any{map[string]any{"id": "a", "size": 1}, map[string]any{"id": "b", "size": 2}}, "nullable": nil}
	after := map[string]any{"items": []any{map[string]any{"id": "b", "size": 3}, map[string]any{"id": "a", "size": 1}}, "flag": false}
	changes := Diff(before, after, []IdentityRule{{Path: "/items", KeyPath: "/id"}})
	require.Len(t, changes, 5)
	require.Equal(t, "add", changes[0].Op)
	require.Equal(t, false, changes[0].After)
	require.Equal(t, "move", changes[1].Op)
	require.Equal(t, "b", changes[1].Identity)
	require.Equal(t, "/items/0/size", changes[2].Path)
	require.Equal(t, "remove", changes[4].Op)
	bad := []any{map[string]any{"id": "a"}, map[string]any{"id": "a"}}
	require.Len(t, Diff(map[string]any{"items": bad}, after, []IdentityRule{{Path: "/items", KeyPath: "/id"}}), 2)
}
func TestJSONPointerAndStrictEncodings(t *testing.T) {
	input := map[string]any{"a/b": map[string]any{"~key": []any{false, nil, ""}}}
	v, ok := Pointer(input, "/a~1b/~0key/0")
	require.True(t, ok)
	require.Equal(t, false, v)
	_, ok = Pointer(input, "/a~1b/~0key/01")
	require.False(t, ok)
	_, ok = Pointer(input, "/a~2b")
	require.False(t, ok)
	_, err := Encode("false", "boolean")
	require.Error(t, err)
	s, err := Encode(false, "boolean")
	require.NoError(t, err)
	require.Equal(t, "false", s)
}

func TestResolutionBoundsExpansionAndUnsafeNumbers(t *testing.T) {
	model := Model{Dialect: Dialect, Schema: map[string]any{"type": "object", "properties": map[string]any{"number": map[string]any{"type": "number"}}}}
	_, _, errs := Resolve(model, map[string]any{"number": float64(9007199254740992)})
	require.Equal(t, "InputComplexity", errs[0].Code)
	definitions := map[string]any{"leaf": map[string]any{"type": "string"}}
	previous := "leaf"
	for i := range 18 {
		name := fmt.Sprintf("level%d", i)
		definitions[name] = map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"$ref": "#/definitions/" + previous}, "b": map[string]any{"$ref": "#/definitions/" + previous}}}
		previous = name
	}
	model.Schema = map[string]any{"type": "object", "definitions": definitions, "properties": map[string]any{"root": map[string]any{"$ref": "#/definitions/" + previous}}}
	_, _, errs = Resolve(model, map[string]any{})
	require.Equal(t, "InvalidSchema", errs[0].Code)
	require.Contains(t, errs[0].Message, "10000")
}
