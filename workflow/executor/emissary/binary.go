package emissary

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func copyBinary() error {
	name, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get the argoexec path: %w", err)
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
