package sso

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v3/config"

	pkgrand "github.com/argoproj/pkg/rand"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/argoproj/argo-workflows/v3/server/auth/types"
)

const (
	Prefix                              = "Bearer v2:"
	issuer                              = "argo-server"                // the JWT issuer
	secretName                          = "sso"                        // where we store SSO secret
	cookieEncryptionPrivateKeySecretKey = "cookieEncryptionPrivateKey" // the key name for the private key in the secret
)

//go:generate mockery --name=Interface

type Interface interface {
	Authorize(authorization string) (*types.Claims, error)
	HandleRedirect(writer http.ResponseWriter, request *http.Request)
	HandleCallback(writer http.ResponseWriter, request *http.Request)
	IsRBACEnabled() bool
}

var _ Interface = &sso{}

type Config = config.SSOConfig

type sso struct {
	config          *oauth2.Config
	issuer          string
	idTokenVerifier *oidc.IDTokenVerifier
	httpClient      *http.Client
	baseHRef        string
	secure          bool
	privateKey      crypto.PrivateKey
	encrypter       jose.Encrypter
	rbacConfig      *config.RBACConfig
	expiry          time.Duration
	customClaimName string
	userInfoPath    string
}

func (s *sso) IsRBACEnabled() bool {
	return s.rbacConfig.IsEnabled()
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
	// Create http client with TLSConfig to allow skipping of CA validation if InsecureSkipVerify is set.
	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify}}}
	oidcContext := oidc.ClientContext(ctx, httpClient)
	// Some offspec providers like Azure, Oracle IDCS have oidc discovery url different from issuer url which causes issuerValidation to fail
	// This providerCtx will allow the Verifier to succeed if the alternate/alias URL is in the config
	if c.IssuerAlias != "" {
		oidcContext = oidc.InsecureIssuerURLContext(oidcContext, c.IssuerAlias)
	}

	provider, err := factory(oidcContext, c.Issuer)
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
	lf := log.Fields{"redirectUrl": config.RedirectURL, "issuer": c.Issuer, "issuerAlias": "DISABLED", "clientId": c.ClientID, "scopes": config.Scopes, "insecureSkipVerify": c.InsecureSkipVerify}
	if c.IssuerAlias != "" {
		lf["issuerAlias"] = c.IssuerAlias
	}
	log.WithFields(lf).Info("SSO configuration")

	return &sso{
		config:          config,
		idTokenVerifier: idTokenVerifier,
		baseHRef:        baseHRef,
		httpClient:      httpClient,
		secure:          secure,
		privateKey:      privateKey,
		encrypter:       encrypter,
		rbacConfig:      c.RBAC,
		expiry:          c.GetSessionExpiry(),
		customClaimName: c.CustomGroupClaimName,
		userInfoPath:    c.UserInfoPath,
		issuer:          c.Issuer,
	}, nil
}

func (s *sso) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	redirectUrl := r.URL.Query().Get("redirect")
	state, err := pkgrand.RandString(10)
	if err != nil {
		log.WithError(err).Error("failed to create state")
		w.WriteHeader(500)
		return
	}
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
		return
	}
	redirectOption := oauth2.SetAuthURLParam("redirect_uri", s.getRedirectUrl(r))
	// Use sso.httpClient in order to respect TLSOptions
	oauth2Context := context.WithValue(ctx, oauth2.HTTPClient, s.httpClient)
	oauth2Token, err := s.config.Exchange(oauth2Context, r.URL.Query().Get("code"), redirectOption)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		w.WriteHeader(401)
		return
	}
	idToken, err := s.idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	c := &types.Claims{}
	if err := idToken.Claims(c); err != nil {
		w.WriteHeader(401)
		return
	}

	// Default to groups claim but if customClaimName is set
	// extract groups based on that claim key
	groups := c.Groups
	if s.customClaimName != "" {
		groups, err = c.GetCustomGroup(s.customClaimName)
		if err != nil {
			w.WriteHeader(401)
			return
		}
	}

	// Some SSO implementations (Okta) require a call to
	// the OIDC user info path to get attributes like groups
	if s.userInfoPath != "" {
		groups, err = c.GetUserInfoGroups(oauth2Token.AccessToken, s.issuer, s.userInfoPath)
		if err != nil {
			w.WriteHeader(401)
			return
		}
	}

	argoClaims := &types.Claims{
		Claims: jwt.Claims{
			Issuer:  issuer,
			Subject: c.Subject,
			Expiry:  jwt.NewNumericDate(time.Now().Add(s.expiry)),
		},
		Groups:             groups,
		RawClaim:           c.RawClaim,
		Email:              c.Email,
		EmailVerified:      c.EmailVerified,
		ServiceAccountName: c.ServiceAccountName,
		PreferredUsername:  c.PreferredUsername,
	}

	raw, err := jwt.Encrypted(s.encrypter).Claims(argoClaims).CompactSerialize()
	if err != nil {
		w.WriteHeader(401)
		return
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
