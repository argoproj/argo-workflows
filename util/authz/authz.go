package authz

import (
	"github.com/casbin/casbin/v2"
)

func NewEnforcer(dirname string) (casbin.IEnforcer, error) {
	e, err := casbin.NewEnforcer(dirname+"/model.conf", dirname+"/policy.csv", Logger)
	if err != nil {
		return nil, err
	}
	e.AddFunction("contains", containsFunc)
	return e, nil
}
