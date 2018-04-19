package ftp

import (
	"github.com/argoproj/argo/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
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

func (f *FTPArtifactDriver) newSFTPClient() (*sftp.Client, error) {
	config := &ssh.ClientConfig{
		User: f.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(f.Password),
		},
		Config: ssh.Config{
			Ciphers: []string{"3des-cbc", "aes256-cbc", "aes192-cbc", "aes128-cbc", "aes192-ctr", "aes256-ctr"},
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", f.Endpoint, config)
  if err != nil {
		return nil, errors.Errorf(errors.CodeNotFound, "Could not dial SFTP server")
  }
  sftpClient, err := sftp.NewClient(conn)
  if err != nil {
		return nil, errors.Errorf(errors.CodeNotFound, "Erroc creating SFTP client")
	}
	
	return sftpClient, nil
}

func (f *FTPArtifactDriver) newFTPClient() (*goftp.FTP, error) {
	ftpClient, err := goftp.Connect(f.Endpoint)
  if err != nil {
		return nil, errors.Errorf(errors.CodeNotFound, "FTP server not available")
	}

	return ftpClient, ftpClient.Login(f.Username, f.Password)
}

// Load artifacts from a SFTP / FTP server
func (f *FTPArtifactDriver) Load(inputArtifact *wfv1.Artifact, path string) error {
	if f.Secure {
		sftpClient, err := f.newSFTPClient()
		if err != nil {
			return err
		}
		defer sftpClient.Close()

		file, err := sftpClient.Open(f.File)
		if err != nil {
			return errors.Errorf(errors.CodeNotFound, "FTP server not available")
		}
		defer file.Close()
	
		tmp, err := os.Create(path)
		if err != nil {
			return errors.Errorf(errors.CodeNotFound, "FTP server not available")
		}
		defer tmp.Close()
	
		if _, err = io.Copy(tmp, file); err != nil {
			return errors.Errorf(errors.CodeNotFound, "FTP server not available")
		}
	
		return tmp.Sync()
	} else {
		ftpClient, err := f.newFTPClient()
		if err != nil {
			return err
		}
	
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
}

// Save artifacts to a SFTP / FTP server
func (f *FTPArtifactDriver) Save(path string, outputArtifact *wfv1.Artifact) error {
	if f.Secure {
		sftpClient, err := f.newSFTPClient()
		if err != nil {
			return err
		}
		defer sftpClient.Close()

		_, err = sftpClient.Create(path)
		if err != nil {
			return errors.Errorf(errors.CodeInternal, err.Error())
		}
	} else {
		ftpClient, err := f.newFTPClient()
		if err != nil {
			return err
		}
		defer ftpClient.Close()

		// Upload artifact
		var file *os.File
		if file, err = os.Open(path); err != nil {
			return errors.Errorf(errors.CodeInternal, err.Error())
		}

		return ftpClient.Stor(outputArtifact.Name, file)
	}

	return nil
}

// Username / password authentication
func login(ftpCLient *goftp.FTP, username, password string) error {
	return ftpCLient.Login(username, password)
}
