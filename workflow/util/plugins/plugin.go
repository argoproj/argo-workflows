package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v3/util/errors"
)

type Plugin struct {
	Address string
	invalid map[string]bool
}

func New(address string) Plugin {
	return Plugin{
		Address: address,
		invalid: map[string]bool{},
	}
}

var client = http.Client{
	Timeout: 3 * time.Second,
}

func (p *Plugin) Call(method string, args interface{}, reply interface{}) error {
	if p.invalid[method] {
		return nil
	}
	body, err := json.Marshal(args)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/%s", p.Address, method), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		return json.NewDecoder(resp.Body).Decode(reply)
	case 404:
		log.WithField("address", p.Address).
			WithField("method", method).
			Info("method not found, never calling again")
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
}
