package parameterization

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// HistoryPager signs positions in an already verified, committed parent chain.
// It stores no parameter content and never substitutes for caller-scoped reads.
// Servers behind the same endpoint should share a randomly generated 32-byte
// key. An ephemeral key is sufficient for a single local server; restarting it
// expires cursors and requires restarting the history query, not saving again.
type HistoryPager struct {
	key []byte
	now func() time.Time
}

var processHistoryPager = sync.OnceValues(func() (*HistoryPager, error) { return NewHistoryPager(nil) })

func NewHistoryPager(key []byte) (*HistoryPager, error) {
	if key == nil {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("history cursor key must contain exactly 32 random bytes")
	}
	return &HistoryPager{key: append([]byte(nil), key...), now: time.Now}, nil
}

type historyPosition struct {
	Version       int    `json:"v"`
	Namespace     string `json:"ns"`
	Set           string `json:"set"`
	SetUID        string `json:"uid"`
	Anchor        string `json:"anchor"`
	AnchorUID     string `json:"anchorUID"`
	AnchorVersion string `json:"anchorVersion"`
	Next          string `json:"next"`
	Offset        int64  `json:"offset"`
	Expires       int64  `json:"expires"`
	// Brent's constant-space cycle detector carries only an ancestor name and
	// counters, instead of putting an ever-growing revision list in the cursor.
	Checkpoint string `json:"checkpoint"`
	Power      int64  `json:"power"`
	Distance   int64  `json:"distance"`
}

func historyExpired() error {
	return apierrors.NewResourceExpired("history cursor expired or changed; reload history from the first page")
}

func (p *HistoryPager) sign(position historyPosition) string {
	raw, _ := json.Marshal(position)
	mac := hmac.New(sha256.New, p.key)
	_, _ = mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (p *HistoryPager) verify(token string) (historyPosition, error) {
	var position historyPosition
	if len(token) > 4096 {
		return position, historyExpired()
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return position, historyExpired()
	}
	raw, e1 := base64.RawURLEncoding.DecodeString(parts[0])
	sig, e2 := base64.RawURLEncoding.DecodeString(parts[1])
	mac := hmac.New(sha256.New, p.key)
	_, _ = mac.Write(raw)
	if e1 != nil || e2 != nil || !hmac.Equal(sig, mac.Sum(nil)) || json.Unmarshal(raw, &position) != nil || position.Version != 1 || position.Expires <= p.now().Unix() || position.Next == "" || position.Offset < 0 || position.Power < 1 || position.Distance < 0 || position.Distance >= position.Power {
		return historyPosition{}, historyExpired()
	}
	return position, nil
}

func (p *HistoryPager) page(ctx context.Context, s Store, name string, limit int64, token string) (*unstructured.UnstructuredList, error) {
	if limit < 1 || limit > 200 {
		return nil, fmt.Errorf("history limit must be between 1 and 200")
	}
	set, err := s.get(ctx, "parametersets", name)
	if err != nil {
		return nil, err
	}
	position := historyPosition{Version: 1, Namespace: s.Namespace, Set: name, SetUID: string(set.GetUID()), Power: 1}
	if token != "" {
		position, err = p.verify(token)
		if err != nil {
			return nil, err
		}
		if position.Namespace != s.Namespace || position.Set != name || position.SetUID != string(set.GetUID()) {
			return nil, historyExpired()
		}
	} else {
		position.Anchor, _, _ = unstructured.NestedString(set.Object, "spec", "revision")
		position.Next = position.Anchor
		position.Checkpoint = position.Anchor
	}
	out := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{}}
	if position.Anchor == "" {
		return out, nil
	}
	anchor, err := s.get(ctx, "parameterrevisions", position.Anchor)
	if err != nil {
		return nil, err
	}
	if token != "" && (position.AnchorUID != string(anchor.GetUID()) || position.AnchorVersion != anchor.GetResourceVersion()) {
		return nil, historyExpired()
	}
	position.AnchorUID = string(anchor.GetUID())
	position.AnchorVersion = anchor.GetResourceVersion()
	for position.Next != "" && int64(len(out.Items)) < limit {
		if position.Offset >= 1000000 {
			return nil, fmt.Errorf("history exceeds one million revisions; archive older versions before continuing")
		}
		obj := anchor
		if position.Next != position.Anchor {
			obj, err = s.get(ctx, "parameterrevisions", position.Next)
			if err != nil {
				return nil, err
			}
		}
		var revision Revision
		if err = decode(obj, &revision); err != nil {
			return nil, err
		}
		if revision.Set != name {
			return nil, fmt.Errorf("revision set mismatch")
		}
		out.Items = append(out.Items, *obj)
		position.Next = revision.Parent
		position.Offset++
		position.Distance++
		if position.Next != "" && position.Next == position.Checkpoint {
			return nil, fmt.Errorf("revision cycle")
		}
		if position.Distance == position.Power {
			position.Checkpoint = position.Next
			position.Power *= 2
			position.Distance = 0
		}
	}
	if position.Next != "" {
		position.Expires = p.now().Add(15 * time.Minute).Unix()
		out.SetContinue(p.sign(position))
	}
	return out, nil
}
