package http

import (
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
)

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
		isSSE := req.Header.Get("Accept") == "text/event-stream"
		if isSSE {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Transfer-Encoding", "chunked")
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}
		runtime.ForwardResponseStream(ctx, mux, marshaler, w, req, recv, opts...)
	}
)
