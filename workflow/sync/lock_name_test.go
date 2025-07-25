package sync

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestDecodeLockName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	type args struct {
		lockName string
	}
	tests := []struct {
		name    string
		args    args
		want    *lockName
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"TestMutexLockNameValidation",
			args{"default/Mutex/test"},
			&lockName{
				Namespace:    "default",
				ResourceName: "test",
				Key:          "",
				Kind:         lockKindMutex,
			},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
		{
			"TestMutexLocksCanContainSlashes",
			args{"default/Mutex/test/foo/bar/baz"},
			&lockName{
				Namespace:    "default",
				ResourceName: "test/foo/bar/baz",
				Key:          "",
				Kind:         lockKindMutex,
			},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
		{
			"TestDatabaseLockNamesWork",
			args{"default/Database/foo"},
			&lockName{
				Namespace:    "default",
				ResourceName: "foo",
				Kind:         lockKindDatabase,
			},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
		{
			"TestConfigMapLockNamesWork",
			args{"default/ConfigMap/foo/bar"},
			&lockName{
				Namespace:    "default",
				ResourceName: "foo",
				Key:          "bar",
				Kind:         lockKindConfigMap,
			},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
		{
			"TestConfigMapKeysCannotContainSlashes",
			args{"default/ConfigMap/foo/bar/baz/qux"},
			nil,
			func(t assert.TestingT, err error, i ...interface{}) bool {
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
			assert.Equalf(t, tt.want, got, "decodeLockName(%v)", tt.args.lockName)
			got.validateEncoding(ctx, tt.args.lockName)
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
