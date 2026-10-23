package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func TestResourcesRegisteredAndDeepCopied(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, AddToScheme(scheme))
	for _, kind := range []string{"ParameterTemplate", "ParameterSet", "ParameterRevision", "WorkflowForm", "WorkflowParameterBinding", "WorkflowSubmission", "ActionDefinition", "ActionResultTemplate", "WorkflowActionBinding", "ParameterRetry", "ParameterNodeInput", "ResourceOperation"} {
		for _, name := range []string{kind, kind + "List"} {
			obj, err := scheme.New(SchemeGroupVersion.WithKind(name))
			require.NoError(t, err)
			clone := obj.DeepCopyObject()
			require.Equal(t, obj, clone)
			require.NotSame(t, obj, clone)
		}
	}
	original := &ParameterSet{Spec: ParameterSetSpec{Template: "model", Values: JSONValue{Raw: json.RawMessage(`{"enabled":false,"count":0,"unset":null,"nested":[1,2]}`)}}}
	copied := original.DeepCopy()
	copied.Spec.Values.Raw[0] = '['
	require.Equal(t, byte('{'), original.Spec.Values.Raw[0])
}

func TestStrictResourceDecoding(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, AddToScheme(scheme))
	codecs := serializer.NewCodecFactory(scheme, serializer.EnableStrict)
	valid := `{"apiVersion":"parameters.workflows.example.io/v1alpha1","kind":"WorkflowParameterBinding","metadata":{"name":"sample"},"spec":{"workflowTemplate":"native","parameterTemplate":"model","mappings":[{"path":"/zones","parameter":"zones","encoding":"json"}]}}`
	obj, _, err := codecs.UniversalDeserializer().Decode([]byte(valid), nil, nil)
	require.NoError(t, err)
	require.Equal(t, "zones", obj.(*WorkflowParameterBinding).Spec.Mappings[0].Parameter)
	for _, input := range []string{
		`{"apiVersion":"parameters.workflows.example.io/v1alpha1","kind":"WorkflowParameterBinding","spec":{"workflowTemplate":123}}`,
		`{"apiVersion":"parameters.workflows.example.io/v1alpha1","kind":"WorkflowParameterBinding","spec":{"workflowTemplte":"typo"}}`,
		`{"apiVersion":"parameters.workflows.example.io/v1alpha1","kind":"WorkflowForm","spec":{"options":{"/region":{"kind":"configmap","allowManual":"true"}}}}`,
	} {
		_, _, err = codecs.UniversalDeserializer().Decode([]byte(input), nil, nil)
		require.Error(t, err)
	}
}

func TestJSONValueDeepCopyPreservesNilAndEmpty(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, {}, json.RawMessage(`{"nested":[false,0,null]}`)} {
		original := &JSONValue{Raw: raw}
		copied := original.DeepCopy()
		require.Equal(t, original, copied)
		if len(raw) > 0 {
			copied.Raw[0] = '['
			require.Equal(t, byte('{'), original.Raw[0])
		}
	}
}

func TestDynamicValuesRetainTypes(t *testing.T) {
	for _, value := range []string{`false`, `0`, `null`, `["a",2,null]`, `{"x":1,"extra":{"nested":false}}`, `"text"`} {
		source := []byte(`{"metadata":{},"spec":{"template":"model","values":` + value + `}}`)
		var set ParameterSet
		require.NoError(t, json.Unmarshal(source, &set))
		raw, err := json.Marshal(set)
		require.NoError(t, err)
		var back ParameterSet
		require.NoError(t, json.Unmarshal(raw, &back))
		require.JSONEq(t, value, string(back.Spec.Values.Raw))
	}
}
