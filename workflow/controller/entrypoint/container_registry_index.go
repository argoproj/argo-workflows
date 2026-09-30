package entrypoint

import (
	"context"
	"fmt"
	"runtime"

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	pkgv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type containerRegistryIndex struct {
	kubernetesClient kubernetes.Interface
}

func (i *containerRegistryIndex) Lookup(ctx context.Context, image string, options Options) (*Image, error) {
	kc, err := k8schain.New(ctx, i.kubernetesClient, k8schain.Options{
		Namespace:          options.Namespace,
		ServiceAccountName: options.ServiceAccountName,
		ImagePullSecrets:   imagePullSecretNames(options.ImagePullSecrets),
	})
	if err != nil {
		return nil, err
	}
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}
	var defaultPlatform = pkgv1.Platform{
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}
	desc, err := remote.Get(ref, remote.WithAuthFromKeychain(kc))
	if err != nil {
		return nil, err
	}
	img, err := imageFromDescriptor(ref, desc, defaultPlatform)
	if err != nil {
		return nil, err
	}
	f, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}
	return &Image{
		Entrypoint: f.Config.Entrypoint,
		Cmd:        f.Config.Cmd,
	}, nil
}

// imageFromDescriptor resolves a manifest to the image whose config carries the entrypoint and cmd.
// For an index (manifest list) the child matching the controller's own platform is preferred, but any
// other image child is accepted: ENTRYPOINT and CMD are the same for every architecture of an image, and
// the controller's architecture says nothing about where the pod will run (#16258).
func imageFromDescriptor(ref name.Reference, desc *remote.Descriptor, platform pkgv1.Platform) (pkgv1.Image, error) {
	if !desc.MediaType.IsIndex() {
		return desc.Image()
	}
	idx, err := desc.ImageIndex()
	if err != nil {
		return nil, err
	}
	manifest, err := idx.IndexManifest()
	if err != nil {
		return nil, err
	}
	var fallback *pkgv1.Hash
	for _, m := range manifest.Manifests {
		if m.MediaType.IsIndex() {
			continue
		}
		// go-containerregistry treats a child without a platform as linux/amd64
		p := pkgv1.Platform{Architecture: "amd64", OS: "linux"}
		if m.Platform != nil {
			p = *m.Platform
		}
		if p.Satisfies(platform) {
			return idx.Image(m.Digest)
		}
		// buildx attestation manifests are attached with platform unknown/unknown and carry no image config
		if fallback == nil && p.OS != "unknown" && p.Architecture != "unknown" {
			digest := m.Digest
			fallback = &digest
		}
	}
	if fallback == nil {
		return nil, fmt.Errorf("no image manifest in index %s", ref)
	}
	return idx.Image(*fallback)
}

func imagePullSecretNames(secrets []v1.LocalObjectReference) []string {
	var v []string
	for _, s := range secrets {
		v = append(v, s.Name)
	}
	return v
}
