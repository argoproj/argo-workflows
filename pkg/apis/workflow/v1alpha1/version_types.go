package v1alpha1

type Version struct {
	Version      string `json:"version" protobuf:"bytes,1,opt,name=version"`
	BuildDate    string `json:"buildDate" protobuf:"bytes,2,opt,name=buildDate"`
	GitCommit    string `json:"gitCommit" protobuf:"bytes,3,opt,name=gitCommit"`
	GitTag       string `json:"gitTag" protobuf:"bytes,4,opt,name=gitTag"`
	GitTreeState string `json:"gitTreeState" protobuf:"bytes,5,opt,name=gitTreeState"`
	GoVersion    string `json:"goVersion" protobuf:"bytes,6,opt,name=goVersion"`
	Compiler     string `json:"compiler" protobuf:"bytes,7,opt,name=compiler"`
	Platform     string `json:"platform" protobuf:"bytes,8,opt,name=platform"`
}
