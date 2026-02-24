package emissary

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func copyBinary() error {
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
	out, err := os.OpenFile(common.VarRunArgoPath+"/argoexec", os.O_RDWR|os.O_CREATE, 0o555)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
