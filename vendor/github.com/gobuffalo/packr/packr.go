package packr

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"sync"
)

var gil = &sync.Mutex{}
var data = map[string]map[string][]byte{}

// PackBytes packs bytes for a file into a box.
func PackBytes(box string, name string, bb []byte) {
	gil.Lock()
	defer gil.Unlock()
	if _, ok := data[box]; !ok {
		data[box] = map[string][]byte{}
	}
	data[box][name] = bb
}

// PackBytesGzip packets the gzipped compressed bytes into a box.
func PackBytesGzip(box string, name string, bb []byte) error {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(bb)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	PackBytes(box, name, buf.Bytes())
	return nil
}

// PackJSONBytes packs JSON encoded bytes for a file into a box.
func PackJSONBytes(box string, name string, jbb string) error {
	var bb []byte
	err := json.Unmarshal([]byte(jbb), &bb)
	if err != nil {
		return err
	}
	PackBytes(box, name, bb)
	return nil
}
