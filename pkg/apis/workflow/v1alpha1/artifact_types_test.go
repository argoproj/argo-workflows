package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
	"testing"
)

func TestArtifactLocation_IsArchiveLogs(t *testing.T) {
	var l *ArtifactLocation
	assert.False(t, l.IsArchiveLogs())
	assert.False(t, (&ArtifactLocation{}).IsArchiveLogs())
	assert.False(t, (&ArtifactLocation{ArchiveLogs: pointer.BoolPtr(false)}).IsArchiveLogs())
	assert.True(t, (&ArtifactLocation{ArchiveLogs: pointer.BoolPtr(true)}).IsArchiveLogs())
}

func TestArtifactLocation_HasLocation(t *testing.T) {
	var l *ArtifactLocation
	assert.False(t, l.HasLocation(), "Nil")
}

func TestArtifactoryArtifact(t *testing.T) {
	a := &ArtifactoryArtifact{URL: "http://my-host"}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "http://my-host/my-key", a.URL)
	assert.Equal(t, "/my-key", key, "has leading slash")
}

func TestGitArtifact(t *testing.T) {
	a := &GitArtifact{Repo: "my-repo"}
	assert.True(t, a.HasLocation())
	assert.Error(t, a.SetKey("my-key"))
	_, err := a.GetKey()
	assert.Error(t, err)
}

func TestGCSArtifact(t *testing.T) {
	a := &GCSArtifact{Key: "my-key", GCSBucket: GCSBucket{Bucket: "my-bucket"}}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "my-key", key)
}

func TestHDFSArtifact(t *testing.T) {
	a := &HDFSArtifact{HDFSConfig: HDFSConfig{Addresses: []string{"my-address"}}}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "my-key", a.Path)
	assert.Equal(t, "my-key", key)
}

func TestHTTPArtifact(t *testing.T) {
	a := &HTTPArtifact{URL: "http://my-host"}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "http://my-host/my-key", a.URL)
	assert.Equal(t, "/my-key", key, "has leading slack")
}

func TestOSSArtifact(t *testing.T) {
	a := &OSSArtifact{Key: "my-key", OSSBucket: OSSBucket{Endpoint: "my-endpoint", Bucket: "my-bucket"}}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "my-key", key)
}

func TestRawArtifact(t *testing.T) {
	a := &RawArtifact{Data: "my-data"}
	assert.True(t, a.HasLocation())
	assert.Error(t, a.SetKey("my-key"))
	_, err := a.GetKey()
	assert.Error(t, err)
}

func TestS3Artifact(t *testing.T) {
	a := &S3Artifact{Key: "my-key", S3Bucket: S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"}}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "my-key", key)
}

func TestArtifactLocation_Relocate(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		var l *ArtifactLocation
		assert.EqualError(t, l.Relocate(nil), "template artifact location not set")
		assert.Error(t, l.Relocate(&ArtifactLocation{}))
		assert.Error(t, (&ArtifactLocation{}).Relocate(nil))
		assert.Error(t, (&ArtifactLocation{}).Relocate(&ArtifactLocation{}))
		assert.Error(t, (&ArtifactLocation{}).Relocate(&ArtifactLocation{S3: &S3Artifact{}}))
		assert.Error(t, (&ArtifactLocation{S3: &S3Artifact{}}).Relocate(&ArtifactLocation{}))
	})
	t.Run("HasLocation", func(t *testing.T) {
		l := &ArtifactLocation{S3: &S3Artifact{S3Bucket: S3Bucket{Bucket: "my-bucket", Endpoint: "my-endpoint"}, Key: "my-key"}}
		assert.NoError(t, l.Relocate(&ArtifactLocation{S3: &S3Artifact{S3Bucket: S3Bucket{Bucket: "other-bucket"}}}))
		assert.Equal(t, "my-endpoint", l.S3.Endpoint, "endpoint is unchanged")
		assert.Equal(t, "my-bucket", l.S3.Bucket, "bucket is unchanged")
		assert.Equal(t, "my-key", l.S3.Key, "key is unchanged")
	})
	t.Run("NotHasLocation", func(t *testing.T) {
		l := &ArtifactLocation{S3: &S3Artifact{Key: "my-key"}}
		assert.NoError(t, l.Relocate(&ArtifactLocation{S3: &S3Artifact{S3Bucket: S3Bucket{Bucket: "my-bucket"}, Key: "other-key"}}))
		assert.Equal(t, "my-bucket", l.S3.Bucket, "bucket copied from argument")
		assert.Equal(t, "my-key", l.S3.Key, "key is unchanged")
	})
}

func TestArtifactLocation_Get(t *testing.T) {
	var l *ArtifactLocation
	assert.Nil(t, l.Get())
	assert.Nil(t, (&ArtifactLocation{}).Get())
	assert.IsType(t, &GitArtifact{}, (&ArtifactLocation{Git: &GitArtifact{}}).Get())
	assert.IsType(t, &GCSArtifact{}, (&ArtifactLocation{GCS: &GCSArtifact{}}).Get())
	assert.IsType(t, &HDFSArtifact{}, (&ArtifactLocation{HDFS: &HDFSArtifact{}}).Get())
	assert.IsType(t, &HTTPArtifact{}, (&ArtifactLocation{HTTP: &HTTPArtifact{}}).Get())
	assert.IsType(t, &OSSArtifact{}, (&ArtifactLocation{OSS: &OSSArtifact{}}).Get())
	assert.IsType(t, &RawArtifact{}, (&ArtifactLocation{Raw: &RawArtifact{}}).Get())
	assert.IsType(t, &S3Artifact{}, (&ArtifactLocation{S3: &S3Artifact{}}).Get())
}

func TestArtifactLocation_SetType(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.Error(t, l.SetType(nil))
	})
	t.Run("Artifactory", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&ArtifactoryArtifact{}))
		assert.NotNil(t, l.Artifactory)
	})
	t.Run("GCS", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&GCSArtifact{}))
		assert.NotNil(t, l.GCS)
	})
	t.Run("HDFS", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&HDFSArtifact{}))
		assert.NotNil(t, l.HDFS)
	})
	t.Run("HTTP", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&HTTPArtifact{}))
		assert.NotNil(t, l.HTTP)
	})
	t.Run("OSS", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&OSSArtifact{}))
		assert.NotNil(t, l.OSS)
	})
	t.Run("Raw", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&RawArtifact{}))
		assert.NotNil(t, l.Raw)
	})
	t.Run("S3", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&S3Artifact{}))
		assert.NotNil(t, l.S3)
	})
}

func TestArtifactLocation_Key(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var l *ArtifactLocation
		assert.False(t, l.HasKey())
		_, err := l.GetKey()
		assert.Error(t, err, "cannot get nil")
		err = l.SetKey("my-file")
		assert.Error(t, err, "cannot set nil")
	})
	t.Run("Empty", func(t *testing.T) {
		// unlike nil, empty is actually invalid
		l := &ArtifactLocation{}
		assert.False(t, l.HasKey())
		_, err := l.GetKey()
		assert.Error(t, err, "cannot get empty")
		err = l.SetKey("my-file")
		assert.Error(t, err, "cannot set empty")
	})
	t.Run("Artifactory", func(t *testing.T) {
		l := &ArtifactLocation{Artifactory: &ArtifactoryArtifact{URL: "http://my-host/my-dir?a=1"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "http://my-host/my-dir/my-file?a=1", l.Artifactory.URL, "appends to Artifactory path")
	})
	t.Run("Git", func(t *testing.T) {
		l := &ArtifactLocation{Git: &GitArtifact{}}
		assert.False(t, l.HasKey())
		_, err := l.GetKey()
		assert.Error(t, err)
		err = l.SetKey("my-file")
		assert.Error(t, err, "cannot set Git key")
	})
	t.Run("GCS", func(t *testing.T) {
		l := &ArtifactLocation{GCS: &GCSArtifact{Key: "my-dir"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "my-dir/my-file", l.GCS.Key, "appends to GCS key")
	})
	t.Run("HDFS", func(t *testing.T) {
		l := &ArtifactLocation{HDFS: &HDFSArtifact{Path: "my-path"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "my-path/my-file", l.HDFS.Path, "appends to HDFS path")
	})
	t.Run("HTTP", func(t *testing.T) {
		l := &ArtifactLocation{HTTP: &HTTPArtifact{URL: "http://my-host/my-dir?a=1"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "http://my-host/my-dir/my-file?a=1", l.HTTP.URL, "appends to HTTP URL path")
	})
	t.Run("OSS", func(t *testing.T) {
		l := &ArtifactLocation{OSS: &OSSArtifact{Key: "my-dir"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "my-dir/my-file", l.OSS.Key, "appends to OSS key")
	})
	t.Run("Raw", func(t *testing.T) {
		l := &ArtifactLocation{Raw: &RawArtifact{}}
		assert.False(t, l.HasKey())
		_, err := l.GetKey()
		assert.Error(t, err, "cannot get raw key")
		err = l.SetKey("my-file")
		assert.Error(t, err, "cannot set raw key")
	})
	t.Run("S3", func(t *testing.T) {
		l := &ArtifactLocation{S3: &S3Artifact{Key: "my-dir"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "my-dir/my-file", l.S3.Key, "appends to S3 key")
	})
}

func TestArtifactRepositoryRef_GetConfigMapOr(t *testing.T) {
	var r *ArtifactRepositoryRef
	assert.Equal(t, "my-cm", r.GetConfigMapOr("my-cm"))
	assert.Equal(t, "my-cm", (&ArtifactRepositoryRef{}).GetConfigMapOr("my-cm"))
	assert.Equal(t, "my-cm", (&ArtifactRepositoryRef{ConfigMap: "my-cm"}).GetConfigMapOr(""))
}

func TestArtifactRepositoryRef_GetKeyOr(t *testing.T) {
	var r *ArtifactRepositoryRef
	assert.Equal(t, "", r.GetKeyOr(""))
	assert.Equal(t, "my-key", (&ArtifactRepositoryRef{}).GetKeyOr("my-key"))
	assert.Equal(t, "my-key", (&ArtifactRepositoryRef{Key: "my-key"}).GetKeyOr(""))
}

func TestArtifactRepositoryRef_String(t *testing.T) {
	var l *ArtifactRepositoryRef
	assert.Equal(t, "nil", l.String())
	assert.Equal(t, "#", (&ArtifactRepositoryRef{}).String())
	assert.Equal(t, "my-cm#my-key", (&ArtifactRepositoryRef{ConfigMap: "my-cm", Key: "my-key"}).String())
}

func TestArtifactRepositoryRefStatus_String(t *testing.T) {
	var l *ArtifactRepositoryRefStatus
	assert.Equal(t, "nil", l.String())
	assert.Equal(t, "/#", (&ArtifactRepositoryRefStatus{}).String())
	assert.Equal(t, "default-artifact-repository", (&ArtifactRepositoryRefStatus{Default: true}).String())
	assert.Equal(t, "my-ns/my-cm#my-key", (&ArtifactRepositoryRefStatus{Namespace: "my-ns", ArtifactRepositoryRef: ArtifactRepositoryRef{ConfigMap: "my-cm", Key: "my-key"}}).String())
}

func TestArtifact_GetArchive(t *testing.T) {
	assert.NotNil(t, (&Artifact{}).GetArchive())
	assert.Equal(t, &ArchiveStrategy{None: &NoneStrategy{}}, (&Artifact{Archive: &ArchiveStrategy{None: &NoneStrategy{}}}).GetArchive())
}
