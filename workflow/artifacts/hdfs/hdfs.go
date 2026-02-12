package hdfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/keytab"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/file"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/resource"
)

// ArtifactDriver is a driver for HDFS
type ArtifactDriver struct {
	Addresses              []string // comma-separated name nodes
	Path                   string
	Force                  bool
	HDFSUser               string
	KrbOptions             *KrbOptions
	DataTransferProtection string
}

var _ common.ArtifactDriver = &ArtifactDriver{}

// KrbOptions is options for Kerberos
type KrbOptions struct {
	CCacheOptions        *CCacheOptions
	KeytabOptions        *KeytabOptions
	Config               string
	ServicePrincipalName string
}

// CCacheOptions is options for ccache
type CCacheOptions struct {
	CCache credentials.CCache
}

// KeytabOptions is options for keytab
type KeytabOptions struct {
	Keytab   keytab.Keytab
	Username string
	Realm    string
}

// ValidateArtifact validates HDFS artifact
func ValidateArtifact(errPrefix string, art *wfv1.HDFSArtifact) error {
	if len(art.Addresses) == 0 {
		return errors.Errorf(errors.CodeBadRequest, "%s.addresses is required", errPrefix)
	}
	if art.Path == "" {
		return errors.Errorf(errors.CodeBadRequest, "%s.path is required", errPrefix)
	}
	if !filepath.IsAbs(art.Path) {
		return errors.Errorf(errors.CodeBadRequest, "%s.path must be a absolute file path", errPrefix)
	}

	hasKrbCCache := art.KrbCCacheSecret != nil
	hasKrbKeytab := art.KrbKeytabSecret != nil

	if art.HDFSUser == "" && !hasKrbCCache && !hasKrbKeytab {
		return errors.Errorf(errors.CodeBadRequest, "either %s.hdfsUser, %s.krbCCacheSecret or %s.krbKeytabSecret is required", errPrefix, errPrefix, errPrefix)
	}
	if hasKrbKeytab && (art.KrbServicePrincipalName == "" || art.KrbConfigConfigMap == nil || art.KrbUsername == "" || art.KrbRealm == "") {
		return errors.Errorf(errors.CodeBadRequest, "%s.krbServicePrincipalName, %s.krbConfigConfigMap, %s.krbUsername and %s.krbRealm are required with %s.krbKeytabSecret", errPrefix, errPrefix, errPrefix, errPrefix, errPrefix)
	}
	if hasKrbCCache && (art.KrbServicePrincipalName == "" || art.KrbConfigConfigMap == nil) {
		return errors.Errorf(errors.CodeBadRequest, "%s.krbServicePrincipalName and %s.krbConfigConfigMap are required with %s.krbCCacheSecret", errPrefix, errPrefix, errPrefix)
	}

	return nil
}

// CreateDriver constructs ArtifactDriver
func CreateDriver(ctx context.Context, ci resource.Interface, art *wfv1.HDFSArtifact) (*ArtifactDriver, error) {
	var krbConfig string
	var krbOptions *KrbOptions
	var err error

	if art.KrbConfigConfigMap != nil && art.KrbConfigConfigMap.Name != "" {
		krbConfig, err = ci.GetConfigMapKey(ctx, art.KrbConfigConfigMap.Name, art.KrbConfigConfigMap.Key)
		if err != nil {
			return nil, err
		}
	}
	if art.KrbCCacheSecret != nil && art.KrbCCacheSecret.Name != "" {
		bytes, err := ci.GetSecret(ctx, art.KrbCCacheSecret.Name, art.KrbCCacheSecret.Key)
		if err != nil {
			return nil, err
		}
		ccache := new(credentials.CCache)
		err = ccache.Unmarshal([]byte(bytes))
		if err != nil {
			return nil, err
		}
		krbOptions = &KrbOptions{
			CCacheOptions: &CCacheOptions{
				CCache: *ccache,
			},
			Config:               krbConfig,
			ServicePrincipalName: art.KrbServicePrincipalName,
		}
	}
	if art.KrbKeytabSecret != nil && art.KrbKeytabSecret.Name != "" {
		bytes, err := ci.GetSecret(ctx, art.KrbKeytabSecret.Name, art.KrbKeytabSecret.Key)
		if err != nil {
			return nil, err
		}
		ktb := keytab.New()
		err = ktb.Unmarshal([]byte(bytes))
		if err != nil {
			return nil, err
		}
		krbOptions = &KrbOptions{
			KeytabOptions: &KeytabOptions{
				Keytab:   *ktb,
				Username: art.KrbUsername,
				Realm:    art.KrbRealm,
			},
			Config:               krbConfig,
			ServicePrincipalName: art.KrbServicePrincipalName,
		}
	}

	driver := ArtifactDriver{
		Addresses:              art.Addresses,
		Path:                   art.Path,
		Force:                  art.Force,
		HDFSUser:               art.HDFSUser,
		KrbOptions:             krbOptions,
		DataTransferProtection: art.DataTransferProtection,
	}
	return &driver, nil
}

// Load downloads artifacts from HDFS compliant storage
func (driver *ArtifactDriver) Load(ctx context.Context, _ *wfv1.Artifact, path string) error {
	hdfscli, err := createHDFSClient(driver.Addresses, driver.HDFSUser, driver.DataTransferProtection, driver.KrbOptions)
	if err != nil {
		return err
	}
	defer hdfscli.Close()

	srcStat, err := hdfscli.Stat(driver.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New(errors.CodeNotFound, err.Error())
		}
		return err
	}
	if srcStat.IsDir() {
		return fmt.Errorf("HDFS artifact does not suppot directory copy")
	}

	_, err = os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if os.IsNotExist(err) {
		dirPath := filepath.Dir(driver.Path)
		if dirPath != "." && dirPath != "/" {
			// Follow umask for the permission
			err = os.MkdirAll(dirPath, 0o777)
			if err != nil {
				return err
			}
		}
	} else if driver.Force {
		err = os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	err = hdfscli.CopyToLocal(driver.Path, path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New(errors.CodeNotFound, err.Error())
		}
		return err
	}
	return nil
}

func (driver *ArtifactDriver) OpenStream(ctx context.Context, a *wfv1.Artifact) (io.ReadCloser, error) {
	// todo: this is a temporary implementation which loads file to disk first
	return common.LoadToStream(ctx, a, driver)
}

// Save saves an artifact to HDFS compliant storage
func (driver *ArtifactDriver) Save(ctx context.Context, path string, outputArtifact *wfv1.Artifact) error {
	hdfscli, err := createHDFSClient(driver.Addresses, driver.HDFSUser, driver.DataTransferProtection, driver.KrbOptions)
	if err != nil {
		return err
	}
	defer hdfscli.Close()

	isDir, err := file.IsDirectory(path)
	if err != nil {
		return err
	}
	if isDir {
		return fmt.Errorf("HDFS artifact does not suppot directory copy")
	}

	_, err = hdfscli.Stat(driver.Path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if os.IsNotExist(err) {
		dirPath := filepath.Dir(driver.Path)
		if dirPath != "." && dirPath != "/" {
			// Follow umask for the permission
			err = hdfscli.MkdirAll(dirPath, 0o777)
			if err != nil {
				return err
			}
		}
	} else if driver.Force {
		err = hdfscli.Remove(driver.Path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return hdfscli.CopyToRemote(path, driver.Path)
}

// Delete is unsupported for the hdfs artifacts
func (driver *ArtifactDriver) Delete(ctx context.Context, s *wfv1.Artifact) error {
	return common.ErrDeleteNotSupported
}

func (driver *ArtifactDriver) ListObjects(ctx context.Context, artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("ListObjects is currently not supported for this artifact type, but it will be in a future version")
}

func (driver *ArtifactDriver) IsDirectory(ctx context.Context, artifact *wfv1.Artifact) (bool, error) {
	return false, errors.New(errors.CodeNotImplemented, "IsDirectory currently unimplemented for HDFS")
}
