package queue

type Item struct {
	Value Prioritizable
	index int
}

func NewItem(x Prioritizable) *Item {
	return &Item{x, 0}
}
