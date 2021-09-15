package casbin

import (
	"context"
	authclaims "github.com/argoproj/argo-workflows/v3/server/auth/types"
	"github.com/casbin/casbin/v2"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"sync"
)


const (
	ActionGet      = "get"
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionDelete   = "delete"
	ActionList     = "list"
	ActionPatch    = "patch"
	ActionWatch    = "watch"
	ActionDeleteCollection   = "deletecollection"
)

type CustomEnforcer struct {
	enforcer casbin.Enforcer
}

var (
	GetClaims func(ctx context.Context) *authclaims.Claims
	instance *CustomEnforcer
	onlyOnce = sync.Once{}
)

func GetCustomEnforcerInstance() *CustomEnforcer {
	onlyOnce.Do(func () {
		log.Debug("Initializing custom enforcer.")

		e, err := casbin.NewEnforcer("casbin/model.conf", "casbin/policy.csv")
		if os.IsNotExist(err) {
			log.WithError(err).Info("Casbin RBAC disabled")
			e.EnableEnforce(false)
		}
		if err != nil {
			log.Fatal(err)
		}

		// TODO - add config
		e.EnableLog(false)
		instance = &CustomEnforcer{enforcer: *e}
	})

	return instance
}

func (e *CustomEnforcer) enforce(ctx context.Context, resource, namespace, name, verb string) error {
	claims := GetClaims(ctx)

	sub := Sub{Sub: "anonymous"}
	if claims != nil {
		sub.Sub = claims.Subject
		sub.Groups = claims.Groups
	}

	obj := Obj{Resource: resource, Namespace: namespace, Name: name}
	act := verb

	logCtx := log.WithFields(log.Fields{"subject": sub.String(), "resource": obj.Resource, "namespace": obj.Namespace, "name": obj.Name,"action": act})

	logCtx.Debug("Enforcing")
	for _, group := range append([]string{sub.Sub}, sub.Groups...) {
		if ok, err := e.enforcer.Enforce(group, obj, act); err != nil {
			return err
		} else if ok {
			logCtx.Info("permission granted")
			return nil
		}
	}

	logCtx.Info("permission denied")
	return status.Error(codes.PermissionDenied, "not allowed")

}