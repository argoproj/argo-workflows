package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

func (p *Plugin) Call(method string, args interface{}, reply interface{}) error {
	if p.invalid[method] {
		return nil
	}
	req, err := json.Marshal(args)
	if err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("%s/api/v1/%s", p.Address, method), "application/json", bytes.NewBuffer(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		return json.NewDecoder(resp.Body).Decode(reply)
	case 404:
		log.WithField("address", p.invalid).
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
