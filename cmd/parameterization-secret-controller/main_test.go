package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartupRejectsInvalidBackendWithoutDisclosingConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		missingFile      bool
	}{
		{"unavailable", "", "backend registration unavailable", true},
		{"oversized", strings.Repeat("x", 65537), "backend registration unavailable", false},
		{"malformed", "synthetic-private-value", "backend registration invalid", false},
		{"insecure origin", `{"address":"http://vault.example","tokenFile":"synthetic-private-value"}`, "registered HTTPS origin", false},
		{"invalid path", `{"address":"https://vault.example","mount":"../secret","entry":"demo","field":"value","tokenFile":"synthetic-private-value"}`, "invalid external secret path", false},
		{"missing credentials", `{"address":"https://vault.example","mount":"secret","entry":"demo"}`, "token file and field", false},
		{"no cluster identity", `{"address":"https://vault.example","mount":"secret","entry":"demo","field":"value","tokenFile":"synthetic-private-value"}`, "in-cluster configuration required", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldFlags, oldArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine, os.Args = oldFlags, oldArgs })
			flag.CommandLine = flag.NewFlagSet(tc.name, flag.ContinueOnError)
			path := filepath.Join(t.TempDir(), "backend.json")
			if !tc.missingFile {
				require.NoError(t, os.WriteFile(path, []byte(tc.body), 0600))
			}
			os.Args = []string{"parameterization-secret-controller", "--namespace=lab", "--index=fixture", "--config=" + path}
			t.Setenv("KUBERNETES_SERVICE_HOST", "")
			t.Setenv("KUBERNETES_SERVICE_PORT", "")
			err := run()
			require.ErrorContains(t, err, tc.want)
			require.NotContains(t, err.Error(), "synthetic-private-value")
		})
	}
}

func TestStartupRequiresAllRegistrationFlags(t *testing.T) {
	for _, args := range [][]string{nil, {"--namespace=lab"}, {"--namespace=lab", "--index=fixture"}, {"--namespace=lab", "--config=fixture.json"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			oldFlags, oldArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine, os.Args = oldFlags, oldArgs })
			flag.CommandLine = flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			os.Args = append([]string{"parameterization-secret-controller"}, args...)
			require.ErrorContains(t, run(), "namespace, index and config are required")
		})
	}
}
