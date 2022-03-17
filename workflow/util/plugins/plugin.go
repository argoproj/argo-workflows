package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/argoproj/argo-workflows/v3/util/errors"
)

type Client struct {
	address string
	token   string
	client  http.Client
	invalid map[string]bool
	backoff wait.Backoff
}

func New(address, token string, timeout time.Duration, backoff wait.Backoff) Client {
	return Client{
		address: address,
		token:   token,
		client: http.Client{
			Timeout: timeout,
		},
		invalid: map[string]bool{},
		backoff: backoff,
	}
}

func (p *Client) Call(ctx context.Context, method string, args interface{}, reply interface{}) error {
	if p.invalid[method] {
		return nil
	}
	log := log.WithField("address", p.address).WithField("method", method)
	body, err := json.Marshal(args)
	if err != nil {
		return err
	}
	return retry.OnError(p.backoff, func(err error) bool {
		log.WithError(err).Debug("Plugin returned error")
		switch e := err.(type) {
		case interface{ Temporary() bool }:
			if e.Temporary() {
				return true
			}
		}
		return strings.Contains(err.Error(), "connection refused")
	}, func() error {
		log.Debug("Calling plugin")
		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/v1/%s", p.address, method), bytes.NewBuffer(body))
		if err != nil {
			return err
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer "+p.token)
		resp, err := p.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		log.WithField("statusCode", resp.StatusCode).Debug("Called plugin")
		switch resp.StatusCode {
		case 200:
			return json.NewDecoder(resp.Body).Decode(reply)
		case 404:
			log.Info("method not found, not calling again")
			p.invalid[method] = true
			_, err := io.Copy(io.Discard, resp.Body)
			return err
		case 503:
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return errors.NewErrTransient(string(data))
		default:
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf("%s: %s", resp.Status, string(data))
		}
	})
}
