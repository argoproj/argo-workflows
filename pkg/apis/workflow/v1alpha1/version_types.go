package v1alpha1

import (
	"errors"
	"regexp"
)

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

var verRe = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)`)

// BrokenDown returns the major, minor and release components
// of the version number, or error if this is not a release
// The error path is considered "normal" in a non-release build.
func (v Version) Components() (string, string, string, error) {
	matches := verRe.FindStringSubmatch(v.Version)
	if matches == nil {
		return ``, ``, ``, errors.New("Not a formal release")
	}
	return matches[1], matches[2], matches[3], nil
}
