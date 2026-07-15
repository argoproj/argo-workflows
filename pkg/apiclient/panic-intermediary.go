package apiclient

import "google.golang.org/grpc/metadata"

type panicIntermediary struct{}

func (w abstractIntermediary) Header() (metadata.MD, error) {
	panic("implement me")
}

func (w abstractIntermediary) Trailer() metadata.MD {
	panic("implement me")
}

func (w abstractIntermediary) CloseSend() error {
	panic("implement me")
}

func (w abstractIntermediary) SendMsg(any) error {
	panic("implement me")
}

func (w abstractIntermediary) RecvMsg(any) error {
	panic("implement me")
}

func (w abstractIntermediary) SetHeader(metadata.MD) error {
	panic("implement me")
}

func (w abstractIntermediary) SendHeader(metadata.MD) error {
	panic("implement me")
}

func (w abstractIntermediary) SetTrailer(metadata.MD) {
	panic("implement me")
}
