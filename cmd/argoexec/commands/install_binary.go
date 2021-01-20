package commands

import (
	"io"
	"os"

	"github.com/argoproj/argo/workflow/util/path"
)

func installBinary() error {
	name, err := path.Search("argoexec")
	if err != nil {
		return err
	}
	in, err := os.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile("/var/argo/argoexec", os.O_RDWR|os.O_CREATE, 0500) // r-x------
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	return err
}
