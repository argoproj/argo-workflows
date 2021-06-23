package emissary

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	out, err := os.OpenFile("/var/run/argo/argoexec", os.O_RDWR|os.O_CREATE, 0o500) // r-x------
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
