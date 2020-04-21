package rbac

import (
	"context"
	"fmt"

	"github.com/casbin/casbin"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/argoproj/argo/server/auth"
)

type Service interface {
	UnaryServerInterceptor() grpc.UnaryServerInterceptor
	StreamServerInterceptor() grpc.StreamServerInterceptor
	Enforce(ctx context.Context, actObj string) error
}

type service struct {
	enforcer *casbin.Enforcer
}

func NewService() Service {
	return &service{enforcer: casbin.NewEnforcer("config/model.conf", "config/policy.csv")}
}

func (s *service) Enforce(ctx context.Context, op op) error {
	user := auth.GetUser(ctx)
	sub := user.Name
	for _, group := range user.Groups {
		_ = s.enforcer.AddRoleForUser(sub, group)
	}
	act, obj := parseActObj(op)
	allowed := s.enforcer.Enforce(sub, obj, act)
	if allowed {
		log.Debug(fmt.Sprintf("%s is allowed to %s %s", sub, act, obj))
		return nil
	}
	return fmt.Errorf("%s not allowed to %s %s", sub, act, obj)
}

func (s *service) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		err := s.Enforce(ctx, parseOp(info.FullMethod))
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (s *service) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := s.Enforce(ss.Context(), parseOp(info.FullMethod))
		if err != nil {
			return err
		}
		return handler(srv, ss)
	}
}
