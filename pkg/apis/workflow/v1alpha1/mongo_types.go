package v1alpha1

// MongoDB is the spec to perform MongoDB operations
type MongoDB struct {
	// URL of the MongoDB
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Database is the name of the MongoDB database
	Database string `json:"database" protobuf:"bytes,2,opt,name=database"`
	// Collection is the name of the MongoDB collection
	Collection string `json:"collection" protobuf:"bytes,3,opt,name=collection"`
	// Operation is the MongoDB operation to perform - insertOne,deleteOne,updateOne
	Operation string `json:"operation" protobuf:"bytes,4,opt,name=operation"`
	// SuccessCondition is an expression if evaluated to true is considered successful
	SuccessCondition string `json:"successCondition,omitempty" protobuf:"bytes,5,opt,name=successCondition"`
	//ID is the ID of the MongoDB document
	ID string `json:"id,omitempty" protobuf:"bytes,6,opt,name=id"`
	// Body is content of the MongoDB document
	Document string `json:"body,omitempty" protobuf:"bytes,7,opt,name=document"`
}
