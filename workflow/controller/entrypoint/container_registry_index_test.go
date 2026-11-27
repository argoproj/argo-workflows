package entrypoint

import (
	"context"
	"io"
	"log"
	"net/http/httptest"
	"net/url"
	"runtime"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	pkgv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

type testManifest struct {
	platform   pkgv1.Platform
	entrypoint []string
}

func newTestImage(t *testing.T, entrypoint []string) pkgv1.Image {
	t.Helper()
	img, err := random.Image(64, 1)
	require.NoError(t, err)
	cfg, err := img.ConfigFile()
	require.NoError(t, err)
	cfg = cfg.DeepCopy()
	cfg.Config.Entrypoint = entrypoint
	cfg.Config.Cmd = []string{"--cmd"}
	img, err = mutate.ConfigFile(img, cfg)
	require.NoError(t, err)
	return img
}

func newTestRegistry(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return u.Host
}

func lookupEntrypoint(t *testing.T, image string) (*Image, error) {
	t.Helper()
	index := &containerRegistryIndex{kubernetesClient: fake.NewSimpleClientset()}
	return index.Lookup(context.Background(), image, Options{Namespace: "default", ServiceAccountName: "default"})
}

func TestContainerRegistryIndexLookupImage(t *testing.T) {
	ref, err := name.ParseReference(newTestRegistry(t) + "/test/image:latest")
	require.NoError(t, err)
	require.NoError(t, remote.Write(ref, newTestImage(t, []string{"/image"})))

	img, err := lookupEntrypoint(t, ref.String())
	require.NoError(t, err)
	require.Equal(t, []string{"/image"}, img.Entrypoint)
	require.Equal(t, []string{"--cmd"}, img.Cmd)
}

func TestContainerRegistryIndexLookupIndex(t *testing.T) {
	native := pkgv1.Platform{OS: runtime.GOOS, Architecture: runtime.GOARCH}
	other := pkgv1.Platform{OS: "linux", Architecture: "arm64"}
	if runtime.GOARCH == "arm64" {
		other.Architecture = "amd64"
	}
	attestation := pkgv1.Platform{OS: "unknown", Architecture: "unknown"}

	tests := map[string]struct {
		manifests []testManifest
		expected  []string
	}{
		"single arch matching controller": {
			manifests: []testManifest{{native, []string{"/native"}}},
			expected:  []string{"/native"},
		},
		"single arch different from controller": {
			manifests: []testManifest{{other, []string{"/other"}}},
			expected:  []string{"/other"},
		},
		"multi arch prefers controller arch": {
			manifests: []testManifest{{other, []string{"/other"}}, {native, []string{"/native"}}},
			expected:  []string{"/native"},
		},
		"skips attestation manifests": {
			manifests: []testManifest{{attestation, []string{"/attestation"}}, {other, []string{"/other"}}},
			expected:  []string{"/other"},
		},
	}
	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			var idx pkgv1.ImageIndex = empty.Index
			for _, m := range tt.manifests {
				idx = mutate.AppendManifests(idx, mutate.IndexAddendum{
					Add:        newTestImage(t, m.entrypoint),
					Descriptor: pkgv1.Descriptor{Platform: &m.platform},
				})
			}
			ref, err := name.ParseReference(newTestRegistry(t) + "/test/index:latest")
			require.NoError(t, err)
			require.NoError(t, remote.WriteIndex(ref, idx))

			img, err := lookupEntrypoint(t, ref.String())
			require.NoError(t, err)
			require.Equal(t, tt.expected, img.Entrypoint)
			require.Equal(t, []string{"--cmd"}, img.Cmd)
		})
	}
}
