package sso

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"time"

	pkgrand "github.com/argoproj/pkg/rand"
	"github.com/coreos/go-oidc"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/argoproj/argo-workflows/v3/server/auth/rbac"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
)

const (
	Prefix                              = "Bearer v2:"
	issuer                              = "argo-server"                // the JWT issuer
	secretName                          = "sso"                        // where we store SSO secret
	cookieEncryptionPrivateKeySecretKey = "cookieEncryptionPrivateKey" // the key name for the private key in the secret
)

//go:generate mockery -name Interface

type Interface interface {
	Authorize(authorization string) (*types.Claims, error)
	HandleRedirect(writer http.ResponseWriter, request *http.Request)
	HandleCallback(writer http.ResponseWriter, request *http.Request)
	IsRBACEnabled() bool
}

var _ Interface = &sso{}

type sso struct {
	config          *oauth2.Config
	idTokenVerifier *oidc.IDTokenVerifier
	baseHRef        string
	secure          bool
	privateKey      crypto.PrivateKey
	encrypter       jose.Encrypter
	rbacConfig      *rbac.Config
	expiry          time.Duration
}

func (s *sso) IsRBACEnabled() bool {
	return s.rbacConfig.IsEnabled()
}

type Config struct {
	Issuer       string                  `json:"issuer"`
	ClientID     apiv1.SecretKeySelector `json:"clientId"`
	ClientSecret apiv1.SecretKeySelector `json:"clientSecret"`
	RedirectURL  string                  `json:"redirectUrl"`
	RBAC         *rbac.Config            `json:"rbac,omitempty"`
	// additional scopes (on top of "openid")
	Scopes        []string        `json:"scopes,omitempty"`
	SessionExpiry metav1.Duration `json:"sessionExpiry,omitempty"`
}

func (c Config) GetSessionExpiry() time.Duration {
	if c.SessionExpiry.Duration > 0 {
		return c.SessionExpiry.Duration
	}
	return 10 * time.Hour
}

// Abstract methods of oidc.Provider that our code uses into an interface. That
// will allow us to implement a stub for unit testing.  If you start using more
// oidc.Provider methods in this file, add them here and provide a stub
// implementation in test.
type providerInterface interface {
	Endpoint() oauth2.Endpoint
	Verifier(config *oidc.Config) *oidc.IDTokenVerifier
}

type providerFactory func(ctx context.Context, issuer string) (providerInterface, error)

func providerFactoryOIDC(ctx context.Context, issuer string) (providerInterface, error) {
	return oidc.NewProvider(ctx, issuer)
}

func New(c Config, secretsIf corev1.SecretInterface, baseHRef string, secure bool) (Interface, error) {
	return newSso(providerFactoryOIDC, c, secretsIf, baseHRef, secure)
}

func newSso(
	factory providerFactory,
	c Config,
	secretsIf corev1.SecretInterface,
	baseHRef string,
	secure bool,
) (Interface, error) {
	if c.Issuer == "" {
		return nil, fmt.Errorf("issuer empty")
	}
	if c.ClientID.Name == "" || c.ClientID.Key == "" {
		return nil, fmt.Errorf("clientID empty")
	}
	if c.ClientSecret.Name == "" || c.ClientSecret.Key == "" {
		return nil, fmt.Errorf("clientSecret empty")
	}
	ctx := context.Background()
	clientSecretObj, err := secretsIf.Get(ctx, c.ClientSecret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	provider, err := factory(context.Background(), c.Issuer)
	if err != nil {
		return nil, err
	}
	var clientIDObj *apiv1.Secret
	if c.ClientID.Name == c.ClientSecret.Name {
		clientIDObj = clientSecretObj
	} else {
		clientIDObj, err = secretsIf.Get(ctx, c.ClientID.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}
	generatedKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	// whoa - are you ignoring errors - yes - we don't care if it fails -
	// if it fails, then the get will fail, and the pod restart
	// it may fail due to race condition with another pod - which is fine,
	// when it restart it'll get the new key
	_, err = secretsIf.Create(ctx, &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName},
		Data:       map[string][]byte{cookieEncryptionPrivateKeySecretKey: x509.MarshalPKCS1PrivateKey(generatedKey)},
	}, metav1.CreateOptions{})
	if err != nil && !apierr.IsAlreadyExists(err) {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}
	secret, err := secretsIf.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(secret.Data[cookieEncryptionPrivateKeySecretKey])
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	clientID := clientIDObj.Data[c.ClientID.Key]
	if clientID == nil {
		return nil, fmt.Errorf("key %s missing in secret %s", c.ClientID.Key, c.ClientID.Name)
	}
	clientSecret := clientSecretObj.Data[c.ClientSecret.Key]
	if clientSecret == nil {
		return nil, fmt.Errorf("key %s missing in secret %s", c.ClientSecret.Key, c.ClientSecret.Name)
	}
	config := &oauth2.Config{
		ClientID:     string(clientID),
		ClientSecret: string(clientSecret),
		RedirectURL:  c.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       append(c.Scopes, oidc.ScopeOpenID),
	}
	idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: config.ClientID})
	encrypter, err := jose.NewEncrypter(jose.A256GCM, jose.Recipient{Algorithm: jose.RSA_OAEP_256, Key: privateKey.Public()}, &jose.EncrypterOptions{Compression: jose.DEFLATE})
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT encrpytor: %w", err)
	}
	log.WithFields(log.Fields{"redirectUrl": config.RedirectURL, "issuer": c.Issuer, "clientId": c.ClientID, "scopes": config.Scopes}).Info("SSO configuration")
	return &sso{
		config:          config,
		idTokenVerifier: idTokenVerifier,
		baseHRef:        baseHRef,
		secure:          secure,
		privateKey:      privateKey,
		encrypter:       encrypter,
		rbacConfig:      c.RBAC,
		expiry:          c.GetSessionExpiry(),
	}, nil
}

func (s *sso) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	redirectUrl := r.URL.Query().Get("redirect")
	state := pkgrand.RandString(10)
	http.SetCookie(w, &http.Cookie{
		Name:     state,
		Value:    redirectUrl,
		Expires:  time.Now().Add(3 * time.Minute),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})

	redirectOption := oauth2.SetAuthURLParam("redirect_uri", s.getRedirectUrl(r))
	http.Redirect(w, r, s.config.AuthCodeURL(state, redirectOption), http.StatusFound)
}

func (s *sso) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(state)
	http.SetCookie(w, &http.Cookie{Name: state, MaxAge: 0})
	if err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(fmt.Sprintf("invalid state: %v", err)))
		return
	}
	redirectOption := oauth2.SetAuthURLParam("redirect_uri", s.getRedirectUrl(r))
	oauth2Token, err := s.config.Exchange(ctx, r.URL.Query().Get("code"), redirectOption)
	if err != nil {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to exchange token: %v", err)))
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		w.WriteHeader(401)
		_, _ = w.Write([]byte("failed to get id_token"))
		return
	}
	idToken, err := s.idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to verify token: %v", err)))
		return
	}
	c := &types.Claims{}
	if err := idToken.Claims(c); err != nil {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to get claims: %v", err)))
		return
	}
	argoClaims := &types.Claims{
		Claims: jwt.Claims{
			Issuer:  issuer,
			Subject: c.Subject,
			Expiry:  jwt.NewNumericDate(time.Now().Add(s.expiry)),
		},
		Groups:             c.Groups,
		Email:              c.Email,
		EmailVerified:      c.EmailVerified,
		ServiceAccountName: c.ServiceAccountName,
	}
	raw, err := jwt.Encrypted(s.encrypter).Claims(argoClaims).CompactSerialize()
	if err != nil {
		panic(err)
	}
	value := Prefix + raw
	log.Debugf("handing oauth2 callback %v", value)
	http.SetCookie(w, &http.Cookie{
		Value:    value,
		Name:     "authorization",
		Path:     s.baseHRef,
		Expires:  time.Now().Add(s.expiry),
		SameSite: http.SameSiteStrictMode,
		Secure:   s.secure,
	})
	redirect := s.baseHRef

	proto := "http"
	if s.secure {
		proto = "https"
	}
	prefix := fmt.Sprintf("%s://%s%s", proto, r.Host, s.baseHRef)

	if strings.HasPrefix(cookie.Value, prefix) {
		redirect = cookie.Value
	}
	http.Redirect(w, r, redirect, 302)
}

// authorize verifies a bearer token and pulls user information form the claims.
func (s *sso) Authorize(authorization string) (*types.Claims, error) {
	tok, err := jwt.ParseEncrypted(strings.TrimPrefix(authorization, Prefix))
	if err != nil {
		return nil, fmt.Errorf("failed to parse encrypted token %v", err)
	}
	c := &types.Claims{}
	if err := tok.Claims(s.privateKey, c); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %v", err)
	}
	if err := c.Validate(jwt.Expected{Issuer: issuer}); err != nil {
		return nil, fmt.Errorf("failed to validate claims: %v", err)
	}
	return c, nil
}

func (s *sso) getRedirectUrl(r *http.Request) string {
	if s.config.RedirectURL != "" {
		return s.config.RedirectURL
	}

	proto := "http"

	if r.URL.Scheme != "" {
		proto = r.URL.Scheme
	} else if s.secure {
		proto = "https"
	}

	return fmt.Sprintf("%s://%s%soauth2/callback", proto, r.Host, s.baseHRef)
}
