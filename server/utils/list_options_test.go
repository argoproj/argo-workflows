package utils

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestListOptionsMethods(t *testing.T) {
	baseOptions := ListOptions{}

	t.Run("WithLimit", func(t *testing.T) {
		result := baseOptions.WithLimit(10)
		require.Equal(t, 10, result.Limit)
	})

	t.Run("WithOffset", func(t *testing.T) {
		result := baseOptions.WithOffset(5)
		require.Equal(t, 5, result.Offset)
	})

	t.Run("WithShowRemainingItemCount", func(t *testing.T) {
		result := baseOptions.WithShowRemainingItemCount(true)
		require.True(t, result.ShowRemainingItemCount)
	})

	t.Run("WithMaxStartedAt", func(t *testing.T) {
		now := time.Now()
		result := baseOptions.WithMaxStartedAt(now)
		require.Equal(t, now, result.MaxStartedAt)
	})

	t.Run("WithMinStartedAt", func(t *testing.T) {
		now := time.Now()
		result := baseOptions.WithMinStartedAt(now)
		require.Equal(t, now, result.MinStartedAt)
	})

	t.Run("WithStartedAtAscending", func(t *testing.T) {
		result := baseOptions.WithStartedAtAscending(true)
		require.True(t, result.StartedAtAscending)
	})
}

func TestBuildListOptions(t *testing.T) {
	tests := []struct {
		name          string
		options       metav1.ListOptions
		ns            string
		namePrefix    string
		expected      ListOptions
		expectedError error
	}{
		{
			name: "Basic case",
			options: metav1.ListOptions{
				Continue: "5",
				Limit:    10,
			},
			ns:         "default",
			namePrefix: "test-",
			expected: ListOptions{
				Namespace:  "default",
				NamePrefix: "test-",
				Limit:      10,
				Offset:     5,
			},
		},
		{
			name: "Invalid continue",
			options: metav1.ListOptions{
				Continue: "invalid",
			},
			expectedError: status.Error(codes.InvalidArgument, "listOptions.continue must be int"),
		},
		{
			name: "Negative continue",
			options: metav1.ListOptions{
				Continue: "-1",
			},
			expectedError: status.Error(codes.InvalidArgument, "listOptions.continue must >= 0"),
		},
		{
			name: "Field selectors",
			options: metav1.ListOptions{
				FieldSelector: "metadata.namespace=test,metadata.name=myname,spec.startedAt>2023-01-01T00:00:00Z,spec.startedAt<2023-12-31T23:59:59Z,ext.showRemainingItemCount=true",
			},
			expected: ListOptions{
				Namespace:              "test",
				Name:                   "myname",
				NameOperator:           "=",
				MinStartedAt:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				MaxStartedAt:           time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
				ShowRemainingItemCount: true,
			},
		},
		{
			name: "Invalid field selector",
			options: metav1.ListOptions{
				FieldSelector: "unsupported=value",
			},
			expectedError: status.Error(codes.InvalidArgument, "unsupported requirement unsupported=value"),
		},
		{
			name: "Conflicting namespace in query param and field selector",
			options: metav1.ListOptions{
				FieldSelector: "metadata.namespace=test",
			},
			ns: "different-namespace",
			expectedError: status.Errorf(codes.InvalidArgument,
				"'namespace' query param (%q) and fieldselector 'metadata.namespace' (%q) are both specified and contradict each other",
				"different-namespace", "test"),
		},
		{
			name: "Unsupported metadata.name field selector",
			options: metav1.ListOptions{
				FieldSelector: "metadata.name:invalid",
			},
			expectedError: status.Errorf(codes.InvalidArgument,
				"unsupported fieldselector 'metadata.name' metadata.name:invalid"),
		},
		{
			name: "Invalid maxStartedAt< format in field selector",
			options: metav1.ListOptions{
				FieldSelector: "spec.startedAt<invalid-date-format",
			},
			expectedError: func() error {
				_, err := time.Parse(time.RFC3339, "invalid-date-format")
				return ToStatusError(err, codes.Internal)
			}(),
		},
		{
			name: "Invalid maxStartedAt> format in field selector",
			options: metav1.ListOptions{
				FieldSelector: "spec.startedAt>invalid-date-format",
			},
			expectedError: func() error {
				_, err := time.Parse(time.RFC3339, "invalid-date-format")
				return ToStatusError(err, codes.Internal)
			}(),
		},
		{
			name: "Invalid showRemainingItemCount in field selector",
			options: metav1.ListOptions{
				FieldSelector: "ext.showRemainingItemCount=invalid",
			},
			expectedError: func() error {
				_, err := strconv.ParseBool("invalid")
				return ToStatusError(err, codes.Internal)
			}(),
		},
		{
			name: "Label selector",
			options: metav1.ListOptions{
				LabelSelector: "app=myapp,env=prod,label==mylabel",
			},
			expected: ListOptions{
				LabelRequirements: mustParseToRequirements(t, "app=myapp,env=prod,label==mylabel"),
			},
		},
		{
			name: "Invalid label selector",
			options: metav1.ListOptions{
				LabelSelector: "app=myapp,",
			},
			expectedError: status.Error(codes.InvalidArgument, "found '', expected: identifier after ','"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildListOptions(tt.options, tt.ns, tt.namePrefix)
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSplitTerm(t *testing.T) {
	tests := []struct {
		name    string
		term    string
		wantOp  string
		wantRhs string
		wantOk  bool
	}{
		{
			name:    "Not equal operator",
			term:    "field!=value",
			wantOp:  "!=",
			wantRhs: "value",
			wantOk:  true,
		},
		{
			name:    "Double equal operator",
			term:    "field==value",
			wantOp:  "==",
			wantRhs: "value",
			wantOk:  true,
		},
		{
			name:    "Equal operator",
			term:    "field=value",
			wantOp:  "=",
			wantRhs: "value",
			wantOk:  true,
		},
		{
			name:    "No operator",
			term:    "fieldvalue",
			wantOp:  "",
			wantRhs: "",
			wantOk:  false,
		},
		{
			name:    "Invalid operator",
			term:    "field:value",
			wantOp:  "",
			wantRhs: "",
			wantOk:  false,
		},
		{
			name:    "Operator at the end",
			term:    "field=",
			wantOp:  "=",
			wantRhs: "",
			wantOk:  true,
		},
		{
			name:    "Multiple operators",
			term:    "field==value!=othervalue",
			wantOp:  "==",
			wantRhs: "value!=othervalue",
			wantOk:  true,
		},
		{
			name:    "Operator in the middle",
			term:    "pre!=post=value",
			wantOp:  "!=",
			wantRhs: "post=value",
			wantOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, rhs, ok := splitTerm(tt.term)
			require.Equal(t, tt.wantOp, op)
			require.Equal(t, tt.wantRhs, rhs)
			require.Equal(t, tt.wantOk, ok)
		})
	}
}

func mustParseToRequirements(t *testing.T, labelSelector string) labels.Requirements {
	requirements, err := labels.ParseToRequirements(labelSelector)
	require.NoError(t, err)
	return requirements
}
