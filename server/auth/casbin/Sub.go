package casbin

import "strings"

type Sub struct {
	Sub string
	Groups []string
}

func (s Sub) String() string {
	return s.Sub + ", " + strings.Join(s.Groups, ",")
}

