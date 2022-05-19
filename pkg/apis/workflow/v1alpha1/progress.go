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

var progressRune = map[NodePhase]rune{
	NodePending:   'P',
	NodeRunning:   'R',
	NodeSucceeded: 'S',
	NodeFailed:    'F',
	NodeSkipped:   'K',
}

var progressPhase = map[rune]NodePhase{
	'P': NodePending,
	'R': NodeRunning,
	'S': NodeSucceeded,
	'F': NodeFailed,
	'K': NodeSkipped,
}

func ParseProgress(s string) (Progress, bool) {
	v := Progress(s)
	return v, v.IsValid()
}

func PendingProgress(len int) Progress {
	return Progress(strings.Repeat("P", len))
}

func (in Progress) parts() []string {
	return strings.SplitN(string(in), "/", 2)
}

func (in Progress) N() int {
	if strings.ContainsRune(string(in), '/') {
		v, _ := strconv.Atoi(in.parts()[0])
		return v
	}
	n := 0
	for _, x := range strings.Split(string(in), "") {
		if x != "P" {
			n++
		}
	}
	return n
}

func (in Progress) M() int {
	if strings.ContainsRune(string(in), '/') {
		v, _ := strconv.Atoi(in.parts()[1])
		return v
	}
	return len(string(in))
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

func (in Progress) Failure() bool {
	return strings.ContainsRune(string(in), progressRune[NodeFailed])
}

func (in Progress) WithStatus(i int, x NodePhase) Progress {
	out := []rune(in)
	out[i] = progressRune[x]
	return Progress(out)
}

func (in Progress) Status(i int) NodePhase {
	if i >= len(in) {
		return NodePending
	}
	return progressPhase[([]rune(in))[i]]
}
