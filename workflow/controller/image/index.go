package image

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/lru"
)

var cache = lru.New(1024)

func Lookup(image string) ([]string, error) {
	if cmd, ok := cache.Get(image); ok {
		log.WithField("image", image).WithField("cmd", cmd).Debug("Cache hit")
		return cmd.([]string), nil
	}
	log.WithField("image", image).Debug("Cache miss")
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, err
	}
	f, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}
	var cmd []string
	if len(f.Config.Entrypoint) > 0 {
		cmd = f.Config.Entrypoint
	} else {
		cmd = f.Config.Cmd
	}
	cache.Add(image, cmd)
	return cmd, nil
}
