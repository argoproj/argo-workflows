package prediction

type nullDurationPredictor struct{}

var NullDurationPredictor DurationPredictor = nullDurationPredictor{}
