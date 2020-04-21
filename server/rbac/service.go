package rbac

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/casbin/casbin"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/argoproj/argo/server/auth"
)

// https://casbin.org/editor/
type Service interface {
	UnaryServerInterceptor() grpc.UnaryServerInterceptor
	StreamServerInterceptor() grpc.StreamServerInterceptor
	Enforce(ctx context.Context, actObj string) error
}

type service struct {
	enforcer *casbin.Enforcer
}

func NewService(policyCsv string) (Service, error) {
	if policyCsv == "" {
		return nil, fmt.Errorf("policyCsv empty")
	}
	err := ioutil.WriteFile("model.conf", []byte(`[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && keyMatch(r.act, p.act)
`), 0666)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile("policy.csv", []byte(policyCsv), 0666)
	if err != nil {
		return nil, err
	}
	enforcer := casbin.NewEnforcer("model.conf", "policy.csv")
	log.WithField("policyCsv", policyCsv).Debug()
	return &service{enforcer: enforcer}, nil
}

func (s *service) isAllowed(sub, obj, act string) (bool, string) {
	allowed := s.enforcer.Enforce(sub, obj, act)
	if allowed {
		return true, fmt.Sprintf("%s is allowed to %s %s", sub, act, obj)
	} else {
		return false, fmt.Sprintf("%s not allowed to %s %s", sub, act, obj)
	}
}

func partFullMethod(fullMethod string) string {
	parts := strings.SplitN(fullMethod, "/", 3)
	// TODO can panic
	return parts[2]
}

func ParseActObj(actObj string) (string, string) {
	for act := range map[string]bool{
		"Create":  true,
		"Delete":  true,
		"Get":     true,
		"List":    true,
		"Lint":    true,
		"PodLogs": true,
		"Suspend": true,
		"Retry":   true,
		"Resume":  true,
		"Update":  true,
		"Watch":   true,
	} {
		if strings.HasPrefix(actObj, act) {
			return strings.ToLower(act), strings.ToLower(strings.TrimPrefix(actObj, act))
		}
	}
	panic("cannot parse " + actObj)
}

func (s *service) Enforce(ctx context.Context, actObj string) error {
	user := auth.GetUser(ctx)
	sub := user.Name
	for _, group := range user.Groups {
		_ = s.enforcer.AddRoleForUser(sub, group)
	}
	act, obj := ParseActObj(actObj)
	allowed, msg := s.isAllowed(sub, obj, act)
	log.Debug(msg)
	if !allowed {
		return fmt.Errorf(msg)
	}
	return nil
}

func (s *service) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		err := s.Enforce(ctx, partFullMethod(info.FullMethod))
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (s *service) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := s.Enforce(ss.Context(), partFullMethod(info.FullMethod))
		if err != nil {
			return err
		}
		return handler(srv, ss)
	}
}
