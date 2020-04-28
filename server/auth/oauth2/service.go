package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/argoproj/pkg/jwt/zjwt"
	"github.com/argoproj/pkg/rand"
	"github.com/coreos/go-oidc"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// https://github.com/dexidp/dex/blob/master/Documentation/using-dex.md

const Prefix = "Bearer id_token:"

type claims struct {
	Groups []string `json:"groups"`
}

type Service interface {
	Authorize(ctx context.Context, authorization string) (wfv1.User, error)
	HandleRedirect(writer http.ResponseWriter, request *http.Request)
	HandleCallback(writer http.ResponseWriter, request *http.Request)
}

type service struct {
	config          *oauth2.Config
	idTokenVerifier *oidc.IDTokenVerifier
	baseHRef        string
	secure          bool
}

func NewService(issuer, clientID, clientSecret, redirectURL, baseHRef string, secure bool) (Service, error) {
	if issuer == "" {
		return nil, fmt.Errorf("issuer empty")
	}
	if clientID == "" {
		return nil, fmt.Errorf("clientId empty")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("clientSecret empty")
	}
	if redirectURL == "" {
		return nil, fmt.Errorf("redirectUrl empty")
	}
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, err
	}
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "groups"},
	}
	idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	log.WithFields(log.Fields{"redirectURL": config.RedirectURL, "issuer": issuer, "clientId": clientID}).Info("SSO configuration")
	return &service{config, idTokenVerifier, baseHRef, secure}, nil
}

const stateCookieName = "oauthState"

func (s *service) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	state := rand.RandString(10)
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Expires:  time.Now().Add(1 * time.Hour),
		HttpOnly: true,
		// TODO - no path?
		// TODO - secure
		// TODO - lax? not strict?
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})
	http.Redirect(w, r, s.config.AuthCodeURL(state), http.StatusFound)
}

func (s *service) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(stateCookieName)
	http.SetCookie(w, &http.Cookie{Name: stateCookieName, MaxAge: 0})
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(fmt.Sprintf("invalid state: %v", err)))
		return
	}
	if state != cookie.Value {
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
	token, err := zjwt.ZJWT(rawIDToken)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to get compress token: %v", err)))
		return
	}
	value := Prefix + token
	log.Debugf("handing oauth2 callback %v", value)
	http.SetCookie(w, &http.Cookie{
		Value:    value,
		Name:     "authorization",
		Path:     s.baseHRef,
		Expires:  time.Now().Add(10 * time.Hour),
		SameSite: http.SameSiteStrictMode,
		Secure:   s.secure,
	})
	http.Redirect(w, r, s.baseHRef, 302)
}

// authorize verifies a bearer token and pulls user information form the claims.
func (s *service) Authorize(ctx context.Context, authorisation string) (wfv1.User, error) {
	rawIDToken, err := zjwt.JWT(strings.TrimPrefix(authorisation, Prefix))
	if err != nil {
		return wfv1.NullUser, fmt.Errorf("failed to decompress token %v", err)
	}
	idToken, err := s.idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return wfv1.NullUser, fmt.Errorf("failed to verify id_token %v", err)
	}
	c := &claims{}
	if err := idToken.Claims(c); err != nil {
		return wfv1.NullUser, fmt.Errorf("failed to parse claims: %v", err)
	}
	return wfv1.User{Name: idToken.Subject, Groups: c.Groups}, nil
}
