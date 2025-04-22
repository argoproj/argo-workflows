package auth

import (
	"context"

	"google.golang.org/grpc"
)

// authorizingServerStream is a thin wrapper around grpc.ServerStream that allows modifying context and do RBAC via gatekeeper.
type authorizingServerStream struct {
	grpc.ServerStream
	ctx context.Context
	Gatekeeper
}

func NewAuthorizingServerStream(ss grpc.ServerStream, gk Gatekeeper) *authorizingServerStream {
	return &authorizingServerStream{
		ServerStream: ss,
		ctx:          ss.Context(),
		Gatekeeper:   gk,
	}
}

func (l *authorizingServerStream) Context() context.Context {
	return l.ctx
}

func (l *authorizingServerStream) SendMsg(m interface{}) error {
	return l.ServerStream.SendMsg(m)
}

// RecvMsg is overridden so that we can understand the request and use it for RBAC
func (l *authorizingServerStream) RecvMsg(m interface{}) error {
	err := l.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}
	ctx, err := l.ContextWithRequest(l.ctx, m)
	if err != nil {
		return err
	}
	l.ctx = ctx
	return nil
}
