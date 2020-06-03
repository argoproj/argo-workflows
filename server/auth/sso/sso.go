package sso

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
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const Prefix = "Bearer id_token:"

type claims struct {
	Groups []string `json:"groups"`
}

type Interface interface {
	Authorize(ctx context.Context, authorization string) error
	HandleRedirect(writer http.ResponseWriter, request *http.Request)
	HandleCallback(writer http.ResponseWriter, request *http.Request)
}

var _ Interface = &sso{}

type sso struct {
	config          *oauth2.Config
	idTokenVerifier *oidc.IDTokenVerifier
	baseHRef        string
	secure          bool
}

type Config struct {
	Issuer       string                  `json:"issuer"`
	ClientID     string                  `json:"clientId"`
	ClientSecret apiv1.SecretKeySelector `json:"clientSecret"`
	RedirectURL  string                  `json:"redirectUrl"`
}

func New(c Config, secretsIf corev1.SecretInterface, baseHRef string, secure bool) (Interface, error) {
	if c.Issuer == "" {
		return nil, fmt.Errorf("issuer empty")
	}
	if c.ClientID == "" {
		return nil, fmt.Errorf("clientId empty")
	}
	if c.ClientSecret.Name == "" || c.ClientSecret.Key == "" {
		return nil, fmt.Errorf("clientSecret empty")
	}
	if c.RedirectURL == "" {
		return nil, fmt.Errorf("redirectUrl empty")
	}
	secrets, err := secretsIf.Get(c.ClientSecret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	provider, err := oidc.NewProvider(context.Background(), c.Issuer)
	if err != nil {
		return nil, err
	}
	config := &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: string(secrets.Data[c.ClientSecret.Key]),
		RedirectURL:  c.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "groups"},
	}
	idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: config.ClientID})
	log.WithFields(log.Fields{"redirectUrl": config.RedirectURL, "issuer": c.Issuer, "clientId": c.ClientID}).Info("SSO configuration")
	return &sso{config, idTokenVerifier, baseHRef, secure}, nil
}

const stateCookieName = "oauthState"

func (s *sso) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	state := rand.RandString(10)
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Expires:  time.Now().Add(3 * time.Minute),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})
	http.Redirect(w, r, s.config.AuthCodeURL(state), http.StatusFound)
}

func (s *sso) HandleCallback(w http.ResponseWriter, r *http.Request) {
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
func (s *sso) Authorize(ctx context.Context, authorisation string) error {
	rawIDToken, err := zjwt.JWT(strings.TrimPrefix(authorisation, Prefix))
	if err != nil {
		return fmt.Errorf("failed to decompress token %v", err)
	}
	idToken, err := s.idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return fmt.Errorf("failed to verify id_token %v", err)
	}
	c := &claims{}
	if err := idToken.Claims(c); err != nil {
		return fmt.Errorf("failed to parse claims: %v", err)
	}
	return nil
}
