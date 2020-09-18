package queue

type val struct {
	key      string
	priority int32
}

func (t *val) GetKey() string {
	return t.key
}

func (t *val) HigherPriorityThan(x interface{}) bool {
	return t.priority > x.(*val).priority
}

var _ Keyed = &val{}
