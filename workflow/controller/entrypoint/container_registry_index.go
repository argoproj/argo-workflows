package entrypoint

import (
	"context"
	"runtime"
	
	v12 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
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
	var defaultPlatform = v12.Platform{
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(kc), remote.WithPlatform(defaultPlatform))
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

func imagePullSecretNames(secrets []v1.LocalObjectReference) []string {
	var v []string
	for _, s := range secrets {
		v = append(v, s.Name)
	}
	return v
}
