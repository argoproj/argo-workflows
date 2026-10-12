package parameterization

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestActionContractsWireTypesAndRedaction(t *testing.T) {
	schema := map[string]any{"type": "object", "properties": map[string]any{"count": map[string]any{"type": "integer", "minimum": 1}, "enabled": map[string]any{"type": "boolean"}, "text": map[string]any{"type": "string", "pattern": "^allowed$"}}, "required": []any{"count", "enabled", "text"}, "additionalProperties": false}
	encoded, err := FreezeActionContract(schema)
	require.NoError(t, err)
	param := func(n, v string) wfv1.Parameter { x := wfv1.AnyString(v); return wfv1.Parameter{Name: n, Value: &x} }
	good := []wfv1.Parameter{param("count", "1"), param("enabled", "false"), param("text", "allowed")}
	require.NoError(t, ValidateActionValues(encoded, good, nil))
	for _, bad := range []string{"0", "1.5", "\"1\"", "1 true", "private-secret"} {
		v := append([]wfv1.Parameter(nil), good...)
		v[0] = param("count", bad)
		err := ValidateActionValues(encoded, v, nil)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "private-secret")
	}
	require.Error(t, ValidateActionValues(encoded, good[:2], nil))
	require.Error(t, ValidateActionValues(encoded, append(good, param("other", "value")), nil))
	require.Error(t, ValidateActionValues(encoded, append(good, good[0]), nil))
	require.Error(t, ValidateActionValues("not-base64", nil, nil))
	require.NoError(t, ValidateActionValues("", good, nil), "native unbound tasks retain behavior")
	require.Error(t, ValidateActionValues(strings.Repeat("x", 45001), nil, nil))
}
func TestActionResultContractAndSchemaDataIsInert(t *testing.T) {
	schema := map[string]any{"type": "object", "properties": map[string]any{"result": map[string]any{"type": "object", "properties": map[string]any{"ok": map[string]any{"type": "boolean"}}, "required": []any{"ok"}, "description": "{{workflow.parameters.not-an-expression}}"}}, "required": []any{"result"}}
	encoded, err := FreezeActionContract(schema)
	require.NoError(t, err)
	require.NotContains(t, encoded, "{{")
	valid := `{"ok":true}`
	require.NoError(t, ValidateActionValues(encoded, nil, &valid))
	invalid := `{"ok":"private-value"}`
	err = ValidateActionValues(encoded, nil, &invalid)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "private-value")
	require.Error(t, ValidateActionValues(encoded, nil, nil))
	schema["properties"].(map[string]any)["result"] = map[string]any{"$ref": "https://untrusted.invalid/schema"}
	_, err = FreezeActionContract(schema)
	require.Error(t, err)
}
