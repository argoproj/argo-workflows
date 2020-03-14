package v1alpha1

type LoggingFacility struct {
	Name      string                   `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Templates LoggingFacilityTemplates `json:"templates,omitempty" protobuf:"bytes,2,opt,name=templates"`
}

type LoggingFacilityTemplates struct {
	Workflow string `json:"workflow,omitempty" protobuf:"bytes,1,opt,name=workflow"`
	Pod      string `json:"pod,omitempty" protobuf:"bytes,2,opt,name=pod"`
}
