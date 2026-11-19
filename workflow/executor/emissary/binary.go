package emissary

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

// copyBinary copies the argoexec binary on PATH to dir/argoexec. It replaces
// the file atomically, so it can run again over an earlier copy: an init
// container re-run in a recreated pod sandbox finds the read-only argoexec it
// wrote before.
func copyBinary(dir string) error {
	name, err := exec.LookPath("argoexec")
	if err != nil {
		return err
	}
	in, err := os.Open(filepath.Clean(name))
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	// argoexec needs to be executable from non-root user in the main container.
	// Therefore we set permission 0o555 == r-xr-xr-x.
	return osspecific.WriteFileAtomic(dir, "argoexec", in, 0o555)
}
