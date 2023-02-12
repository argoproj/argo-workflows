package v1alpha1

import (
	"fmt"
	"strconv"
	"strings"
)

// Progress in one of two  formats.
// (v1) Is N/M format. N is number of task complete. M is number of tasks.
// (v2) Is PRSFK format. M can be inferred by the length of the string N is the number of runes that are not P.
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

var progressPhase = map[rune]NodePhase{}

func init() {
	for k, v := range progressRune {
		progressPhase[v] = k
	}
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
	if in.isV1() {
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

// isV1 returns true if the progress is in the v1 format
func (in Progress) isV1() bool {
	return strings.ContainsRune(string(in), '/')
}

func (in Progress) M() int {
	if in.isV1() {
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

// WithStatus returns a new progress with the status of the i'th node set to x
func (in Progress) WithStatus(i int, x NodePhase) Progress {
	out := []rune(in)
	out[i] = progressRune[x]
	return Progress(out)
}

// Status returns the status of the i'th node
func (in Progress) Status(i int) NodePhase {
	if i >= len(in) {
		return NodePending
	}
	return progressPhase[([]rune(in))[i]]
}
