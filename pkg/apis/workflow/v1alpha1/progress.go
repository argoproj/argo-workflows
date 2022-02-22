package v1alpha1

import (
	"fmt"
	"strconv"
	"strings"
)

// Progress in N/M format. N is number of task complete. M is number of tasks.
type Progress string

const (
	ProgressUndefined = Progress("")
	ProgressZero      = Progress("0/0") // zero value (not the same as "no progress)
	ProgressDefault   = Progress("0/1")
)

func NewProgress(n, m int64) (Progress, bool) {
	return ParseProgress(fmt.Sprintf("%v/%v", n, m))
}

func ParseProgress(s string) (Progress, bool) {
	v := Progress(s)
	return v, v.IsValid()
}

func (in Progress) parts() []string {
	return strings.SplitN(string(in), "/", 2)
}

func (in Progress) N() int64 {
	return parseInt64(in.parts()[0])
}

func (in Progress) M() int64 {
	return parseInt64(in.parts()[1])
}

func (in Progress) Add(x Progress) Progress {
	return Progress(fmt.Sprintf("%v/%v", in.N()+x.N(), in.M()+x.M()))
}

func (in Progress) Complete() Progress {
	return Progress(fmt.Sprintf("%v/%v", in.M(), in.M()))
}

func (in Progress) IsValid() bool {
	return in != "" && in.N() >= 0 && in.N() <= in.M() && in.M() > 0
}

func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
