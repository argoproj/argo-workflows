package prediction

type DurationPredictor interface{}

type durationPredictor struct{}

var _ DurationPredictor = &durationPredictor{}

func NewDurationPredictor() DurationPredictor {
	return &durationPredictor{}
}
