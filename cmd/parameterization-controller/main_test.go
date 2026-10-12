package main

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartupRequiresExplicitNamespaceAndClusterIdentity(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no namespace", nil, "namespace is required"},
		{"no cluster identity", []string{"--namespace=lab"}, "in-cluster configuration required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldFlags, oldArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine, os.Args = oldFlags, oldArgs })
			flag.CommandLine = flag.NewFlagSet(tc.name, flag.ContinueOnError)
			os.Args = append([]string{"parameterization-controller"}, tc.args...)
			t.Setenv("KUBERNETES_SERVICE_HOST", "")
			t.Setenv("KUBERNETES_SERVICE_PORT", "")
			require.ErrorContains(t, run(), tc.want)
		})
	}
}
