package emissary

import (
	"io"
	"os"

	"github.com/argoproj/argo-workflows/v3/workflow/util/path"
)

func copyBinary() error {
	name, err := path.Search("argoexec")
	if err != nil {
		return err
	}
	in, err := os.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile("/var/run/argo/argoexec", os.O_RDWR|os.O_CREATE, 0500) // r-x------
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
