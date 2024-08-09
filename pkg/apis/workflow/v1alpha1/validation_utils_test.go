package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateWorkflowFieldNames(t *testing.T) {
	tests := map[string]struct {
		names             []string
		isParamOrArtifact bool
		expectedErr       error
	}{
		"no name": {
			names:             []string{""},
			isParamOrArtifact: false,
			expectedErr:       fmt.Errorf("[0].name is required"),
		},
		"invalid workflow field name length": {
			// length > 128
			names:             []string{"this-is-a-super-long-template-name-this-is-a-super-long-template-name-this-is-a-super-long-template-name-this-is-a-super-long-template-name"},
			isParamOrArtifact: false,
			expectedErr:       fmt.Errorf("[0].name: 'this-is-a-super-long-template-name-this-is-a-super-long-template-name-this-is-a-super-long-template-name-this-is-a-super-long-template-name' is invalid: must be no more than 128 characters"),
		},
		"invalid workflow field name": {
			names:             []string{"hello_world"},
			isParamOrArtifact: false,
			expectedErr:       fmt.Errorf("[0].name: 'hello_world' is invalid: name must consist of alpha-numeric characters or '-', and must start with an alpha-numeric character (e.g. My-name1-2, 123-NAME)"),
		},
		"invalid artifact name": {
			names:             []string{"$$"},
			isParamOrArtifact: true,
			expectedErr:       fmt.Errorf("[0].name: '$$' is invalid: Parameter/Artifact name must consist of alpha-numeric characters, '_' or '-' e.g. my_param_1, MY-PARAM-1"),
		},
		"duplicate": {
			names:             []string{"a", "a"},
			isParamOrArtifact: false,
			expectedErr:       fmt.Errorf("[1].name 'a' is not unique"),
		},
		"valid artifact name": {
			names:             []string{"artifact-1"},
			isParamOrArtifact: true,
			expectedErr:       nil,
		},
		"valid template name": {
			names:             []string{"template-1"},
			isParamOrArtifact: false,
			expectedErr:       nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateWorkflowFieldNames(tc.names, tc.isParamOrArtifact)
			if tc.expectedErr != nil {
				require.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestVerifyNoCycles(t *testing.T) {
	tests := map[string]struct {
		depGraph    map[string][]string
		expectedErr error
	}{
		"no cycle": {
			depGraph: map[string][]string{
				"b": {"a"},
				"c": {"a"},
				"d": {"b", "c"},
			},
			expectedErr: nil,
		},
		"has cycle 1": {
			depGraph: map[string][]string{
				"a": {"d"},
				"b": {"a"},
				"c": {"b"},
				"d": {"c"},
			},
			expectedErr: fmt.Errorf("dependency cycle detected: d->c->b->a->d"),
		},
		"has cycle 2": {
			depGraph: map[string][]string{
				"a": {},
				"b": {"a"},
				"c": {"b"},
				"d": {"e"},
				"e": {"d"},
			},
			expectedErr: fmt.Errorf("dependency cycle detected: e->d->e"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateNoCycles(tc.depGraph)
			if tc.expectedErr != nil {
				require.Errorf(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
