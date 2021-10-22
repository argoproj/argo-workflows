package auth

import (
    "context"
    "google.golang.org/grpc"
)

// AuthorizingServerStream is a thin wrapper around grpc.ServerStream that allows modifying context and do RBAC via gatekeeper.
type AuthorizingServerStream struct {
    grpc.ServerStream
    ctx context.Context
    Gatekeeper
}

func NewAuthorizingServerStream(ss grpc.ServerStream, gk Gatekeeper) *AuthorizingServerStream {
    return &AuthorizingServerStream{
        ServerStream: ss,
        ctx: ss.Context(),
        Gatekeeper: gk,
    };
}

func (l *AuthorizingServerStream) Context() context.Context {
    return l.ctx
}

func (l *AuthorizingServerStream) SendMsg(m interface{}) error {
    return l.ServerStream.SendMsg(m)
}

// RecvMsg is overridden so that we can understand the request and use it for RBAC
func (l *AuthorizingServerStream) RecvMsg(m interface{}) error {
    err := l.ServerStream.RecvMsg(m)
    if err != nil {
        return err
    }
    ctx, err := l.Gatekeeper.Context(l.ctx, m)
    if err != nil {
        return err
    }
    l.ctx = ctx
    return nil
}
