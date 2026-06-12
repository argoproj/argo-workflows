package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/klauspost/compress/zstd"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/file"
)

type codec struct {
	name   string
	encode func(wfv1.Nodes) ([]byte, error)
	decode func([]byte) (wfv1.Nodes, error)
}

func marshalJSON(nodes wfv1.Nodes) ([]byte, error) {
	return json.Marshal(nodes)
}

func unmarshalJSON(b []byte) (wfv1.Nodes, error) {
	var nodes wfv1.Nodes
	err := json.Unmarshal(b, &nodes)
	return nodes, err
}

func marshalProto(nodes wfv1.Nodes) ([]byte, error) {
	status := wfv1.WorkflowStatus{Nodes: nodes}
	return status.Marshal()
}

func unmarshalProto(b []byte) (wfv1.Nodes, error) {
	var status wfv1.WorkflowStatus
	if err := status.Unmarshal(b); err != nil {
		return nil, err
	}
	return status.Nodes, nil
}

func newZstdPair(level zstd.EncoderLevel, dict []byte) (*zstd.Encoder, *zstd.Decoder, error) {
	encOpts := []zstd.EOption{zstd.WithEncoderLevel(level)}
	decOpts := []zstd.DOption{}
	if dict != nil {
		encOpts = append(encOpts, zstd.WithEncoderDict(dict))
		decOpts = append(decOpts, zstd.WithDecoderDicts(dict))
	}
	enc, err := zstd.NewWriter(nil, encOpts...)
	if err != nil {
		return nil, nil, err
	}
	dec, err := zstd.NewReader(nil, decOpts...)
	if err != nil {
		return nil, nil, err
	}
	return enc, dec, nil
}

func zstdCodec(name string, level zstd.EncoderLevel, dict []byte,
	marshal func(wfv1.Nodes) ([]byte, error), unmarshal func([]byte) (wfv1.Nodes, error),
) (codec, error) {
	enc, dec, err := newZstdPair(level, dict)
	if err != nil {
		return codec{}, fmt.Errorf("%s: %w", name, err)
	}
	return codec{
		name: name,
		encode: func(nodes wfv1.Nodes) ([]byte, error) {
			b, err := marshal(nodes)
			if err != nil {
				return nil, err
			}
			return enc.EncodeAll(b, nil), nil
		},
		decode: func(b []byte) (wfv1.Nodes, error) {
			raw, err := dec.DecodeAll(b, nil)
			if err != nil {
				return nil, err
			}
			return unmarshal(raw)
		},
	}, nil
}

// buildCodecs returns the codec matrix from the spec, in display order. The
// first codec (json+gzip via util/file, i.e. the current packer path) is the
// baseline that ratios are computed against.
func buildCodecs(ctx context.Context, level zstd.EncoderLevel, jsonDict, protoDict []byte) ([]codec, error) {
	gzipCodec := codec{
		name: "json+gzip",
		encode: func(nodes wfv1.Nodes) ([]byte, error) {
			b, err := marshalJSON(nodes)
			if err != nil {
				return nil, err
			}
			return file.CompressContent(ctx, b), nil
		},
		decode: func(b []byte) (wfv1.Nodes, error) {
			raw, err := file.DecompressContent(ctx, b)
			if err != nil {
				return nil, err
			}
			return unmarshalJSON(raw)
		},
	}

	codecs := []codec{gzipCodec}
	for _, c := range []struct {
		name      string
		dict      []byte
		marshal   func(wfv1.Nodes) ([]byte, error)
		unmarshal func([]byte) (wfv1.Nodes, error)
	}{
		{"json+zstd", nil, marshalJSON, unmarshalJSON},
		{"json+zstd+dict", jsonDict, marshalJSON, unmarshalJSON},
		{"proto+zstd", nil, marshalProto, unmarshalProto},
		{"proto+zstd+dict", protoDict, marshalProto, unmarshalProto},
	} {
		zc, err := zstdCodec(c.name, level, c.dict, c.marshal, c.unmarshal)
		if err != nil {
			return nil, err
		}
		codecs = append(codecs, zc)
	}
	return codecs, nil
}
