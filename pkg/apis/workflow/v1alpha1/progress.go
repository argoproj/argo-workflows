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

func ParseProgress(s string) (Progress, bool) {
	v := Progress(s)
	return v, v.IsValid()
}

func (in Progress) parts() (string, string) {
	// Input without a "/" separator is malformed: m stays empty, so M()
	// returns 0 and IsValid() rejects it.
	n, m, _ := strings.Cut(string(in), "/")
	return n, m
}

func (in Progress) N() int64 {
	n, _ := in.parts()
	return parseInt64(n)
}

func (in Progress) M() int64 {
	_, m := in.parts()
	return parseInt64(m)
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
