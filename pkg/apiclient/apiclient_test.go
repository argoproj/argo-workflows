package apiclient

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpts_String(t *testing.T) {
	require.Equal(t, "(argoServerOpts=(url=,path=,secure=false,insecureSkipVerify=false,http=false),instanceID=)", Opts{}.String())
	require.Equal(t, "(argoServerOpts=(url=,path=,secure=false,insecureSkipVerify=false,http=false),instanceID=my-instanceid)", Opts{InstanceID: "my-instanceid"}.String())
}
