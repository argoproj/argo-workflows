package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
)

type sseMarshaller struct {
	runtime.Marshaler
}

func (m *sseMarshaller) ContentType() string {
	return "text/event-stream"
}

func (m *sseMarshaller) Marshal(v interface{}) ([]byte, error) {
	dataBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("data: %s \n\n", string(dataBytes))), nil
}

var (
	StreamForwarder = func(
		ctx context.Context,
		mux *runtime.ServeMux,
		marshaler runtime.Marshaler,
		w http.ResponseWriter,
		req *http.Request,
		recv func() (proto.Message, error),
		opts ...func(context.Context, http.ResponseWriter, proto.Message) error,
	) {
		if req.Header.Get("Accept") == "text/event-stream" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Transfer-Encoding", "chunked")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			runtime.ForwardResponseStream(ctx, mux, &sseMarshaller{marshaler}, w, req, recv, opts...)
		} else {
			runtime.ForwardResponseStream(ctx, mux, marshaler, w, req, recv, opts...)
		}
	}
)
