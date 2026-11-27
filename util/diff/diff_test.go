package diff

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// countingMarshaler counts how often it is marshalled to JSON.
type countingMarshaler struct{ calls *int }

func (c countingMarshaler) MarshalJSON() ([]byte, error) {
	*c.calls++
	return []byte(`{}`), nil
}

func TestLogChanges(t *testing.T) {
	for _, tc := range []struct {
		level    logging.Level
		marshals int
	}{
		{level: logging.Info, marshals: 0},
		{level: logging.Debug, marshals: 2},
	} {
		t.Run(string(tc.level), func(t *testing.T) {
			calls := 0
			ctx := logging.WithLogger(context.Background(), logging.NewSlogLoggerCustom(tc.level, logging.JSON, io.Discard))
			LogChanges(ctx, countingMarshaler{&calls}, countingMarshaler{&calls})
			assert.Equal(t, tc.marshals, calls)
		})
	}
}
