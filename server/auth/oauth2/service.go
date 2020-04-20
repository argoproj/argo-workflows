package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// https://github.com/dexidp/dex/blob/master/Documentation/using-dex.md

const prefix = "id_token "

type claims struct {
	Groups []string `json:"groups"`
}

type Service struct {
	config          *oauth2.Config
	idTokenVerifier *oidc.IDTokenVerifier
	baseHRef        string
}

func NewService(baseHRef string) (*Service, error) {
	// TODO - from config
	issuer := "http://dex:5556/dex"
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, err
	}
	// TODO - from secret
	clientID := "argo-server"
	config := &oauth2.Config{
		ClientID: clientID,
		// TODO - from secret
		ClientSecret: "ZXhhbXBsZS1hcHAtc2VjcmV0",
		// TODO - from config
		RedirectURL: "http://localhost:2746" + baseHRef + "oauth2/callback",
		Endpoint:    provider.Endpoint(),
		Scopes:      []string{oidc.ScopeOpenID, "groups"},
	}
	idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	log.WithFields(log.Fields{"redirectURL": config.RedirectURL, "issuer": issuer}).Info("SSO configuration")
	return &Service{config, idTokenVerifier, baseHRef}, nil
}

// handleRedirect is used to start an OAuth2 flow with the dex server.
func (s *Service) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, s.config.AuthCodeURL("TODO"), http.StatusFound)
}

func (s *Service) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	state := r.URL.Query().Get("state")
	if state != "TODO" {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(fmt.Sprintf("invalid state: %s", state)))
		return
	}
	oauth2Token, err := s.config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to exchange token: %v", err)))
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to get id_token")))
		return
	}
	idToken, err := s.idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to verify token: %v", err)))
		return
	}
	c := &claims{}
	if err := idToken.Claims(c); err != nil {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to get claims: %v", err)))
		return
	}
	value := prefix + rawIDToken
	log.Debugf("handing oauth2 callback %v", value)
	// TODO "httpsonly" etc
	// TODO we must compress this because we know id_token can be large if you have many groups
	http.SetCookie(w, &http.Cookie{Name: "authorization", Value: value, Path: s.baseHRef, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, s.baseHRef, 302)
}

// authorize verifies a bearer token and pulls user information form the claims.
func (s *Service) Authorize(ctx context.Context, authorisation string) (*v1alpha1.User, error) {
	rawIDToken := strings.TrimPrefix(authorisation, prefix)
	idToken, err := s.idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify id_token %v", err)
	}
	c := &claims{}
	if err := idToken.Claims(c); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %v", err)
	}
	return &v1alpha1.User{Name: idToken.Subject, Groups: c.Groups}, nil
}

func (s *Service) IsSSO(authorization string) bool {
	return strings.HasSuffix(authorization, prefix)
}
