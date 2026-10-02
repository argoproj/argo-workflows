package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func TestCapturedPodUIDWireRoundTrip(t *testing.T) {
	for _, uid := range []string{"", "61efeb43-8f0f-4d31-a2e8-21253d0c5b3b"} {
		t.Run(uid, func(t *testing.T) {
			node := NodeStatus{ID: "workflow", Name: "workflow", Type: NodeTypePod, Phase: NodeSucceeded, CapturedPodUID: uid}
			encoded, err := json.Marshal(node)
			require.NoError(t, err)
			if uid == "" {
				require.NotContains(t, string(encoded), "capturedPodUID")
			}
			var fromJSON NodeStatus
			require.NoError(t, json.Unmarshal(encoded, &fromJSON))
			require.Equal(t, node, fromJSON)

			wire, err := proto.Marshal(&node)
			require.NoError(t, err)
			var fromProto NodeStatus
			require.NoError(t, proto.Unmarshal(wire, &fromProto))
			require.Equal(t, node, fromProto)
		})
	}
}
