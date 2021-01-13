package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	nethttp "net/http"
)

func http(args []string) error {
	switch len(args) {
	case 2:
		url := args[1]
		r, err := nethttp.Get(url)
		if err != nil {
			return err
		}
		if r.StatusCode != 200 {
			return fmt.Errorf("not 200 OK: %v", r.Status)
		}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		println(string(data))
		return nil
	}
	return errors.New("usage: argosay http get url")
}
