package sync

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeLockName(t *testing.T) {
	type args struct {
		lockName string
	}
	tests := []struct {
		name    string
		args    args
		want    *LockName
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"TestMutexLockNameValidation",
			args{"default/Mutex/test"},
			&LockName{
				Namespace:    "default",
				ResourceName: "test",
				Key:          "",
				Kind:         LockKindMutex,
			},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
		{
			"TestMutexLocksCanContainSlashes",
			args{"default/Mutex/test/foo/bar/baz"},
			&LockName{
				Namespace:    "default",
				ResourceName: "test/foo/bar/baz",
				Key:          "",
				Kind:         LockKindMutex,
			},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return true
			},
		},
		{
			"TestConfigMapLockNamesWork",
			args{"default/ConfigMap/foo/bar"},
			&LockName{
				Namespace:    "default",
				ResourceName: "foo",
				Key:          "bar",
				Kind:         LockKindConfigMap,
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
			got, err := DecodeLockName(tt.args.lockName)
			if !tt.wantErr(t, err, fmt.Sprintf("DecodeLockName(%v)", tt.args.lockName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "DecodeLockName(%v)", tt.args.lockName)
			got.ValidateEncoding(tt.args.lockName)
		})
	}
}
