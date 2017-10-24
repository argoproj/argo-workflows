package s3

type S3ArtifactDriver struct {
	Endpoint  string
	AccessKey string
	SecretKey string
}

func (s3 *S3ArtifactDriver) Load(sourceURL string, path string) error {

	return nil
}

func (s3 *S3ArtifactDriver) Save(path string, destURL string) (string, error) {

	return destURL, nil
}
