package workflow

//go:generate mockery -name WorkflowServiceClient

// This can be used whenever you cannot accept an empty string (e.g. due to http.cleanPath changing "//" to "/".
const Any = "*"

func PodName(x string) string {
	if x == Any {
		return ""
	}
	return x
}
