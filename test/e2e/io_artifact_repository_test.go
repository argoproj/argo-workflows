//go:build functional

package e2e

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	TestArtifactRepositoriesManifest = "testdata/io-artifact-repositories.yaml"
)

type IOArtifactRepositorySuite struct {
	fixtures.E2ESuite
}

func (s *IOArtifactRepositorySuite) newMinioClient() *minio.Client {
	// Create the minio client for interacting with the bucket.
	c, err := minio.New("localhost:9000", &minio.Options{
		Creds: credentials.NewStaticV4("admin", "password", ""),
	})
	s.Require().NoError(err)
	return c
}

func (s *IOArtifactRepositorySuite) init(c *minio.Client) {
	// Create the object.
	ctx := logging.TestContext(s.T().Context())
	_, err := c.PutObject(ctx, "my-bucket-2", "input-artifact-repository-hello.txt", bytes.NewReader([]byte("hello")), int64(len("hello")), minio.PutObjectOptions{})
	s.Require().NoError(err)
}

func (s *IOArtifactRepositorySuite) cleanup(c *minio.Client) {
	// Delete the object.
	ctx := logging.TestContext(s.T().Context())
	err := c.RemoveObject(ctx, "my-bucket-3", "output-artifact-repository-hello.txt", minio.RemoveObjectOptions{})
	s.Require().NoError(err)
	err = c.RemoveObject(ctx, "my-bucket-2", "input-artifact-repository-hello.txt", minio.RemoveObjectOptions{})
	s.Require().NoError(err)
}

func (s *IOArtifactRepositorySuite) SetupTest() {
	c := s.newMinioClient()
	s.cleanup(c)
	s.init(c)
}

func (s *IOArtifactRepositorySuite) TearDownTest() {
	c := s.newMinioClient()
	s.cleanup(c)
}

func (s *IOArtifactRepositorySuite) TestIOArtifactRepository_Input() {
	then := s.Given().
		KubectlApply(TestArtifactRepositoriesManifest, fixtures.NoError).
		Workflow("@testdata/io-artifact-repository-input-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then()
	then.ExpectArtifact("-", "main-logs", "my-bucket", func(t *testing.T, object minio.ObjectInfo, err error) {
		require.NoError(t, err)
	})
	then.ExpectContainerLogs("main", func(t *testing.T, logs string) {
		require.Contains(t, logs, "hello")
	})
	then.When().
		Exec("kubectl", []string{"-n", fixtures.Namespace, "delete", "-f", TestArtifactRepositoriesManifest}, fixtures.NoError)
}

func (s *IOArtifactRepositorySuite) TestIOArtifactRepository_Output() {
	then := s.Given().
		KubectlApply(TestArtifactRepositoriesManifest, fixtures.NoError).
		Workflow("@testdata/io-artifact-repository-output-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then()
	then.ExpectArtifact("-", "main-logs", "my-bucket", func(t *testing.T, object minio.ObjectInfo, err error) {
		require.NoError(t, err)
	})
	then.ExpectArtifactByKey("output-artifact-repository-hello.txt", "my-bucket-3", func(t *testing.T, object minio.ObjectInfo, err error) {
		require.NoError(t, err)
	})
	then.When().
		Exec("kubectl", []string{"-n", fixtures.Namespace, "delete", "-f", TestArtifactRepositoriesManifest}, fixtures.NoError)
}

func TestIOArtifactRepositorySuite(t *testing.T) {
	suite.Run(t, new(IOArtifactRepositorySuite))
}
