package v1alpha1

type User struct {
	Name   string   `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Groups []string `json:"groups,omitempty" protobuf:"bytes,2,rep,name=groups"`
}
