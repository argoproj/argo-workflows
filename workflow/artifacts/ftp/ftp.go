package ftp

import (
	"fmt"
	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"gopkg.in/dutchcoders/goftp.v1"
	"io"
	"os"
)

// FTPArtifactDriver is the artifact driver for a FTP URL
type FTPArtifactDriver struct {
	Endpoint string
	File     string
	Secure   bool
	Username string
	Password string
}

func (f *FTPArtifactDriver) newFTPClient() (*goftp.FTP, error) {
	ftpClient, err := goftp.Connect(f.Endpoint)
	if err != nil {
		return nil, errors.Errorf(errors.CodeNotFound, "FTP server not available")
	}

	// Username / password authentication
	return ftpClient, login(ftpClient, f.Username, f.Password)
}

// Load artifacts from a FTP server
func (f *FTPArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	ftpClient, err := f.newFTPClient()
	if err != nil {
		return err
	}

	// Download a file into local memory, and calculate it's sha256 hash
	_, err = ftpClient.Retr(f.File, func(r io.Reader) error {
		tmp, err := os.Create(path)
		if err != nil {
			return err
		}
		defer tmp.Close()

		if _, err = io.Copy(tmp, r); err != nil {
			return err
		}

		return tmp.Sync()
	})

	return err
}

// Save artifacts to a FTP server
func (f *FTPArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	ftpClient, err := f.newFTPClient()
	defer ftpClient.Close()

	// Upload artifact
	var file *os.File
	if file, err = os.Open(fmt.Sprintf("%s", path)); err != nil {
		return errors.Errorf(errors.CodeInternal, err.Error())
	}

	return ftpClient.Stor(outputArtifact.Name, file)
}

// Username / password authentication
func login(ftpCLient *goftp.FTP, username, password string) error {
	return ftpCLient.Login(username, password)
}
