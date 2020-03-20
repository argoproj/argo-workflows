package v1alpha1

var NullUser = User{}

type User struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
}
