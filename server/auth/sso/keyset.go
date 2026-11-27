package sso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"golang.org/x/sync/singleflight"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

const (
	jwksMinRefreshInterval = time.Minute
	jwksRetryInterval      = 5 * time.Second
	jwksMaxAge             = time.Hour
	jwksFetchTimeout       = 10 * time.Second
	jwksMaxBytes           = 1 << 20
)

var asymmetricSigningAlgs = []jose.SignatureAlgorithm{
	jose.RS256, jose.RS384, jose.RS512,
	jose.ES256, jose.ES384, jose.ES512,
	jose.PS256, jose.PS384, jose.PS512,
	jose.EdDSA,
}

// routingSigningAlgs also has HMAC, so that such a token from the issuer is rejected rather than handed to another auth mode.
var routingSigningAlgs = append([]jose.SignatureAlgorithm{jose.HS256, jose.HS384, jose.HS512}, asymmetricSigningAlgs...)

// jwksKeySet is an oidc.KeySet with a rate-limited refresh: oidc.RemoteKeySet refetches the keys
// after every failed verification, which unauthenticated bearer tokens could trigger at will.
type jwksKeySet struct {
	url       string
	client    *http.Client
	now       func() time.Time
	fetches   singleflight.Group
	mu        sync.RWMutex
	keys      []jose.JSONWebKey
	fetchedAt time.Time
	nextFetch time.Time
	fetchErr  error
}

func newJWKSKeySet(url string, client *http.Client) *jwksKeySet {
	return &jwksKeySet{url: url, client: client, now: time.Now}
}

func (k *jwksKeySet) VerifySignature(ctx context.Context, raw string) ([]byte, error) {
	jws, err := jose.ParseSignedCompact(raw, asymmetricSigningAlgs)
	if err != nil {
		return nil, fmt.Errorf("malformed jwt: %w", err)
	}
	kid := jws.Signatures[0].Header.KeyID
	keys, err := k.keysFor(ctx, kid)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		if kid == "" || key.KeyID == kid {
			if payload, err := jws.Verify(&key); err == nil {
				return payload, nil
			}
		}
	}
	return nil, errors.New("failed to verify jwt signature")
}

func (k *jwksKeySet) keysFor(ctx context.Context, kid string) ([]jose.JSONWebKey, error) {
	known, refresh := k.needsRefresh(kid)
	if refresh {
		// detached from the request, so a caller that disconnects can't fail the fetch for everyone
		result := k.fetches.DoChan("", func() (any, error) {
			return nil, k.refresh(context.WithoutCancel(ctx))
		})
		if !known {
			select {
			case <-result:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	if len(k.keys) == 0 && k.fetchErr != nil {
		return nil, k.fetchErr
	}
	return k.keys, nil
}

// needsRefresh returns whether the cached keys include kid, and whether to fetch them again.
func (k *jwksKeySet) needsRefresh(kid string) (bool, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	now := k.now()
	known := len(k.keys) > 0 && (kid == "" || slices.ContainsFunc(k.keys, func(key jose.JSONWebKey) bool { return key.KeyID == kid }))
	stale := now.Sub(k.fetchedAt) >= jwksMaxAge
	return known, !now.Before(k.nextFetch) && (!known || stale)
}

func (k *jwksKeySet) refresh(ctx context.Context) error {
	k.mu.RLock()
	skip := k.now().Before(k.nextFetch)
	k.mu.RUnlock()
	if skip {
		return nil
	}
	keys, err := k.fetch(ctx)
	k.mu.Lock()
	defer k.mu.Unlock()
	now := k.now()
	k.fetchErr = err
	if err != nil {
		k.nextFetch = now.Add(jwksRetryInterval)
		fields := logging.Fields{"jwksURL": k.url}
		if len(k.keys) > 0 {
			fields["keysFetchedAt"] = k.fetchedAt
		}
		logging.RequireLoggerFromContext(ctx).WithFields(fields).WithError(err).Warn(ctx, "failed to fetch the OIDC provider's keys")
		return err
	}
	k.keys, k.fetchedAt, k.nextFetch = keys, now, now.Add(jwksMinRefreshInterval)
	return nil
}

func (k *jwksKeySet) fetch(ctx context.Context) ([]jose.JSONWebKey, error) {
	ctx, cancel := context.WithTimeout(ctx, jwksFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch keys: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch keys: %s", resp.Status)
	}
	var set jose.JSONWebKeySet
	if err := json.NewDecoder(io.LimitReader(resp.Body, jwksMaxBytes)).Decode(&set); err != nil {
		return nil, fmt.Errorf("failed to decode keys: %w", err)
	}
	return slices.DeleteFunc(set.Keys, func(key jose.JSONWebKey) bool { return key.Use == "enc" }), nil
}
