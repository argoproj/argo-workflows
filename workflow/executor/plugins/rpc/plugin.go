package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	plugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
)

type plugin struct {
	address string
}

func New(data map[string]string) (interface{}, error) { //nolint:deadcode,unparam
	return &plugin{address: data["address"]}, nil
}

func main() {
	// main funcs are never called in a Go plugin
}

func (p *plugin) call(method string, args interface{}, reply interface{}) error {
	req, err := json.Marshal(args)
	if err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("%s/%s", p.address, method), "application/json", bytes.NewBuffer(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode < 300:
		return json.NewDecoder(resp.Body).Decode(reply)
	default:
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", string(data))
	}
}

var _ plugins.TemplateExecutor = &plugin{}

func (p *plugin) ExecuteTemplate(args plugins.ExecuteTemplateArgs, reply *plugins.ExecuteTemplateReply) error {
	return p.call("TemplateExecutor.ExecuteTemplate", args, reply)
}
