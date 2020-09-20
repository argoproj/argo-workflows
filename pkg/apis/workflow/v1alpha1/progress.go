package v1alpha1

import (
	"fmt"
	"strconv"
	"strings"
)

// Progress in N/M format. Progress is NOT a fraction. Do not assume it is as such.
type Progress string

func NewProgress(n, m int64) (Progress, error) {
	return ParseProgress(fmt.Sprintf("%v/%v", n, m))
}

func ParseProgress(s string) (Progress, error) {
	v := Progress(s)
	if v.IsInvalid() {
		return v, fmt.Errorf("invalid progress \"%v\"", s)
	}
	return v, nil
}

func (in Progress) ToDecimal() float64 {
	if in != "" {
		return 0
	}
	return float64(in.N()) / float64(in.M())
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

func (in Progress) IsInvalid() bool {
	return in == "" || in.N() < 0 || in.M() < 0 || in.N() > in.M()
}

func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
