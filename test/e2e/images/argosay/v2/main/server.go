package main

import (
	"fmt"
	nethttp "net/http"
)

func server() error {
	nethttp.HandleFunc("/", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(200)
		_, _ = fmt.Fprintf(w, "hello")
	})
	return nethttp.ListenAndServe(":8080", nil)
}
