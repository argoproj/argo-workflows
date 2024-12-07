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
			got, err := DecodeLockName(tt.args.lockName)
			if !tt.wantErr(t, err, fmt.Sprintf("decodeLockName(%v)", tt.args.lockName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "decodeLockName(%v)", tt.args.lockName)
			got.validateEncoding(tt.args.lockName)
		})
	}
}
