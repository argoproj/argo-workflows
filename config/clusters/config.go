package clusters

import (
	"time"

	"k8s.io/client-go/rest"
)

type Config struct {
	Host               string
	APIPath            string
	Username           string
	Password           string
	BearerToken        string
	TLSClientConfig    rest.TLSClientConfig
	UserAgent          string
	DisableCompression bool
	QPS                float32
	Burst              int
	Timeout            time.Duration
}

func (c Config) RestConfig() *rest.Config {
	return &rest.Config{
		Host:               c.Host,
		APIPath:            c.APIPath,
		Username:           c.Username,
		Password:           c.Password,
		BearerToken:        c.BearerToken,
		TLSClientConfig:    c.TLSClientConfig,
		UserAgent:          c.UserAgent,
		DisableCompression: c.DisableCompression,
		QPS:                c.QPS,
		Burst:              c.Burst,
		Timeout:            c.Timeout,
	}
}
