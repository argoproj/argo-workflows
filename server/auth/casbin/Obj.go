package casbin

import "fmt"

type Obj struct {
	Resource  string
	Namespace string
	Name      string
}

func (o Obj) String() string {
	return fmt.Sprintf("%s/%s/%s", o.Resource, o.Namespace, o.Name)
}
