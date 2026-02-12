package logging

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
)

// mockArtifactDriver is a mock implementation of ArtifactDriver for testing
type mockArtifactDriver struct {
	mock.Mock
	common.ArtifactDriver
}

func (m *mockArtifactDriver) Load(ctx context.Context, artifact *wfv1.Artifact, path string) error {
	args := m.Called(ctx, artifact, path)
	return args.Error(0)
}

func (m *mockArtifactDriver) OpenStream(ctx context.Context, artifact *wfv1.Artifact) (io.ReadCloser, error) {
	args := m.Called(ctx, artifact)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *mockArtifactDriver) Save(ctx context.Context, path string, artifact *wfv1.Artifact) error {
	args := m.Called(ctx, path, artifact)
	return args.Error(0)
}

func (m *mockArtifactDriver) SaveStream(ctx context.Context, reader io.Reader, artifact *wfv1.Artifact) error {
	args := m.Called(ctx, reader, artifact)
	return args.Error(0)
}

func (m *mockArtifactDriver) Delete(ctx context.Context, artifact *wfv1.Artifact) error {
	args := m.Called(ctx, artifact)
	return args.Error(0)
}

func (m *mockArtifactDriver) ListObjects(ctx context.Context, artifact *wfv1.Artifact) ([]string, error) {
	args := m.Called(ctx, artifact)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockArtifactDriver) IsDirectory(ctx context.Context, artifact *wfv1.Artifact) (bool, error) {
	args := m.Called(ctx, artifact)
	return args.Bool(0), args.Error(1)
}

func TestNew(t *testing.T) {
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	assert.NotNil(t, loggingDriver)
	assert.IsType(t, &driver{}, loggingDriver)
}

func TestLoggingDriver_Load(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	artifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				Key: "test/key",
			},
		},
	}
	path := "/tmp/test-path"

	mockDriver.On("Load", ctx, artifact, path).Return(nil)

	err := loggingDriver.Load(ctx, artifact, path)
	require.NoError(t, err)

	mockDriver.AssertExpectations(t)
}

func TestLoggingDriver_Load_WithError(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	artifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				Key: "test/key",
			},
		},
	}
	path := "/tmp/test-path"

	expectedErr := assert.AnError
	mockDriver.On("Load", ctx, artifact, path).Return(expectedErr)

	err := loggingDriver.Load(ctx, artifact, path)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)

	mockDriver.AssertExpectations(t)
}

func TestLoggingDriver_OpenStream(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	artifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				Key: "test/key",
			},
		},
	}

	expectedReader := io.NopCloser(bytes.NewReader([]byte("test data")))
	mockDriver.On("OpenStream", ctx, artifact).Return(expectedReader, nil)

	reader, err := loggingDriver.OpenStream(ctx, artifact)
	require.NoError(t, err)
	assert.NotNil(t, reader)
	assert.Equal(t, expectedReader, reader)

	mockDriver.AssertExpectations(t)
}

func TestLoggingDriver_Save(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	artifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				Key: "test/key",
			},
		},
	}
	path := "/tmp/test-path"

	mockDriver.On("Save", ctx, path, artifact).Return(nil)

	err := loggingDriver.Save(ctx, path, artifact)
	require.NoError(t, err)

	mockDriver.AssertExpectations(t)
}

func TestLoggingDriver_SaveStream(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	artifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				Key: "test/key",
			},
		},
	}
	reader := bytes.NewReader([]byte("test data"))

	mockDriver.On("SaveStream", ctx, reader, artifact).Return(nil)

	err := loggingDriver.SaveStream(ctx, reader, artifact)
	require.NoError(t, err)

	mockDriver.AssertExpectations(t)
}

func TestLoggingDriver_Delete(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	artifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				Key: "test/key",
			},
		},
	}

	mockDriver.On("Delete", ctx, artifact).Return(nil)

	err := loggingDriver.Delete(ctx, artifact)
	require.NoError(t, err)

	mockDriver.AssertExpectations(t)
}

func TestLoggingDriver_ListObjects(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	artifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				Key: "test/key",
			},
		},
	}

	expectedList := []string{"file1.txt", "file2.txt"}
	mockDriver.On("ListObjects", ctx, artifact).Return(expectedList, nil)

	list, err := loggingDriver.ListObjects(ctx, artifact)
	require.NoError(t, err)
	assert.Equal(t, expectedList, list)

	mockDriver.AssertExpectations(t)
}

func TestLoggingDriver_IsDirectory(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	artifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				Key: "test/key",
			},
		},
	}

	mockDriver.On("IsDirectory", ctx, artifact).Return(true, nil)

	isDir, err := loggingDriver.IsDirectory(ctx, artifact)
	require.NoError(t, err)
	assert.True(t, isDir)

	mockDriver.AssertExpectations(t)
}

func TestLoggingDriver_MethodsWithContext(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	artifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				Key: "test/key",
			},
		},
	}

	t.Run("All methods should log and delegate to underlying driver", func(t *testing.T) {
		// Set up expectations
		mockDriver.On("Load", ctx, artifact, "/tmp/path").Return(nil)
		mockDriver.On("OpenStream", ctx, artifact).Return(io.NopCloser(bytes.NewReader([]byte("test"))), nil)
		mockDriver.On("Save", ctx, "/tmp/path", artifact).Return(nil)
		mockDriver.On("SaveStream", ctx, mock.Anything, artifact).Return(nil)
		mockDriver.On("Delete", ctx, artifact).Return(nil)
		mockDriver.On("ListObjects", ctx, artifact).Return([]string{}, nil)
		mockDriver.On("IsDirectory", ctx, artifact).Return(false, nil)

		// Execute all methods
		_ = loggingDriver.Load(ctx, artifact, "/tmp/path")
		_, _ = loggingDriver.OpenStream(ctx, artifact)
		_ = loggingDriver.Save(ctx, "/tmp/path", artifact)
		_ = loggingDriver.SaveStream(ctx, bytes.NewReader([]byte("test")), artifact)
		_ = loggingDriver.Delete(ctx, artifact)
		_, _ = loggingDriver.ListObjects(ctx, artifact)
		_, _ = loggingDriver.IsDirectory(ctx, artifact)

		// Verify all expectations were met
		mockDriver.AssertExpectations(t)
	})
}

func TestLoggingDriver_MeasuresDuration(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mockDriver := &mockArtifactDriver{}
	loggingDriver := New(mockDriver)

	artifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				Key: "test/key",
			},
		},
	}

	// Mock Load to take some time
	mockDriver.On("Load", ctx, artifact, "/tmp/path").Run(func(args mock.Arguments) {
		time.Sleep(10 * time.Millisecond)
	}).Return(nil)

	start := time.Now()
	err := loggingDriver.Load(ctx, artifact, "/tmp/path")
	duration := time.Since(start)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)

	mockDriver.AssertExpectations(t)
}
