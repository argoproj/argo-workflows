package cmd

import (
	"flag"
	"strconv"

	"k8s.io/klog/v2"
)

// SetGLogLevel set the glog level for the k8s go-client
// this is taken from argoproj/pkg but uses v2 of klog here
// to be compatible with k8s clients v0.19.x and above
func SetGLogLevel(glogLevel int) {
	klog.InitFlags(nil)
	_ = flag.Set("logtostderr", "true")
	_ = flag.Set("v", strconv.Itoa(glogLevel))
}

func GetGLogLevel() string {
	f := flag.Lookup("v")
	if f == nil {
		return ""
	}
	return f.Value.String()
}
