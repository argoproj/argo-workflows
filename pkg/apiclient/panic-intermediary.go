package apiclient

import "google.golang.org/grpc/metadata"

type panicIntermediary struct {
}

func (w abstractIntermediary) Header() (metadata.MD, error) {
	panic("implement me")
}

func (w abstractIntermediary) Trailer() metadata.MD {
	panic("implement me")
}

func (w abstractIntermediary) CloseSend() error {
	panic("implement me")
}

func (w watchIntermediary) SendMsg(interface{}) error {
	panic("implement me")
}

func (w watchIntermediary) RecvMsg(interface{}) error {
	panic("implement me")
}

func (w watchIntermediary) SetHeader(metadata.MD) error {
	panic("implement me")
}

func (w watchIntermediary) SendHeader(metadata.MD) error {
	panic("implement me")
}

func (w watchIntermediary) SetTrailer(metadata.MD) {
	panic("implement me")
}
