package sync

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestDecodeLockName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	type args struct {
		lockName string
	}
	type expected struct {
		namespace    string
		resourceName string
		key          string
		kind         lockKind
	}
	tests := []struct {
		name    string
		args    args
		want    *expected
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"TestMutexLockNameValidation",
			args{"default/Mutex/test"},
			&expected{
				namespace:    "default",
				resourceName: "test",
				key:          "",
				kind:         lockKindMutex,
			},
			func(t assert.TestingT, err error, i ...any) bool {
				return true
			},
		},
		{
			"TestMutexLocksCanContainSlashes",
			args{"default/Mutex/test/foo/bar/baz"},
			&expected{
				namespace:    "default",
				resourceName: "test/foo/bar/baz",
				key:          "",
				kind:         lockKindMutex,
			},
			func(t assert.TestingT, err error, i ...any) bool {
				return true
			},
		},
		{
			"TestDatabaseLockNamesWork",
			args{"default/Database/foo"},
			&expected{
				namespace:    "default",
				resourceName: "foo",
				kind:         lockKindDatabase,
			},
			func(t assert.TestingT, err error, i ...any) bool {
				return true
			},
		},
		{
			"TestConfigMapLockNamesWork",
			args{"default/ConfigMap/foo/bar"},
			&expected{
				namespace:    "default",
				resourceName: "foo",
				key:          "bar",
				kind:         lockKindConfigMap,
			},
			func(t assert.TestingT, err error, i ...any) bool {
				return true
			},
		},
		{
			"TestConfigMapKeysCannotContainSlashes",
			args{"default/ConfigMap/foo/bar/baz/qux"},
			nil,
			func(t assert.TestingT, err error, i ...any) bool {
				return err == nil // this should error
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeLockName(ctx, tt.args.lockName)
			if !tt.wantErr(t, err, fmt.Sprintf("decodeLockName(%v)", tt.args.lockName)) {
				return
			}
			if tt.want != nil {
				assert.Equalf(t, tt.want.namespace, got.GetNamespace(), "decodeLockName(%v).GetNamespace()", tt.args.lockName)
				assert.Equalf(t, tt.want.resourceName, got.GetResourceName(), "decodeLockName(%v).GetResourceName()", tt.args.lockName)
				assert.Equalf(t, tt.want.key, got.GetKey(), "decodeLockName(%v).GetKey()", tt.args.lockName)
				assert.Equalf(t, tt.want.kind, got.getKind(), "decodeLockName(%v).getKind()", tt.args.lockName)
				// Verify encoding roundtrip
				assert.Equalf(t, tt.args.lockName, got.String(ctx), "decodeLockName(%v).String()", tt.args.lockName)
			}
		})
	}
}

func TestNeedDBSession(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tests := []struct {
		name     string
		lockKeys []string
		want     bool
		wantErr  bool
	}{
		{
			name: "NoDatabaseLocks",
			lockKeys: []string{
				"default/ConfigMap/foo/bar",
				"default/Mutex/test",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "SingleDatabaseLock",
			lockKeys: []string{
				"default/Database/foo",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "MixedLockTypesWithDatabase",
			lockKeys: []string{
				"default/ConfigMap/foo/bar",
				"default/Database/foo",
				"default/Mutex/test",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "InvalidLockName",
			lockKeys: []string{
				"default/Invalid/foo",
			},
			want:    false,
			wantErr: true,
		},
		{
			name:     "EmptyLockKeys",
			lockKeys: []string{},
			want:     false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := needDBSession(ctx, tt.lockKeys)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equalf(t, tt.want, got, "needDBS(%v)", tt.lockKeys)
		})
	}
}
