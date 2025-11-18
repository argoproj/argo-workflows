package sso

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v3/config"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"golang.org/x/oauth2"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/argoproj/argo-workflows/v3/server/auth/types"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	pkgrand "github.com/argoproj/argo-workflows/v3/util/rand"
)

const (
	Prefix                              = "Bearer v2:"
	issuer                              = "argo-server"                // the JWT issuer
	secretName                          = "sso"                        // where we store SSO secret
	cookieEncryptionPrivateKeySecretKey = "cookieEncryptionPrivateKey" // the key name for the private key in the secret
)

// Copied from https://github.com/oauth2-proxy/oauth2-proxy/blob/ab448cf38e7c1f0740b3cc2448284775e39d9661/pkg/app/redirect/validator.go#L14-L16
// Used to check final redirects are not susceptible to open redirects.
// Matches //, /\ and both of these with whitespace in between (eg / / or / \).
var invalidRedirectRegex = regexp.MustCompile(`[/\\](?:[\s\v]*|\.{1,2})[/\\]`)

type Interface interface {
	Authorize(authorization string) (*types.Claims, error)
	HandleRedirect(writer http.ResponseWriter, request *http.Request)
	HandleCallback(writer http.ResponseWriter, request *http.Request)
	IsRBACEnabled() bool
}

var _ Interface = &sso{}

type Config = config.SSOConfig

type sso struct {
	config            *oauth2.Config
	issuer            string
	idTokenVerifier   *oidc.IDTokenVerifier
	httpClient        *http.Client
	baseHRef          string
	secure            bool
	privateKey        crypto.PrivateKey
	encrypter         jose.Encrypter
	rbacConfig        *config.RBACConfig
	expiry            time.Duration
	customClaimName   string
	userInfoPath      string
	filterGroupsRegex []*regexp.Regexp
	logger            logging.Logger
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

func New(ctx context.Context, c Config, secretsIf corev1.SecretInterface, baseHRef string, secure bool) (Interface, error) {
	return newSso(ctx, providerFactoryOIDC, c, secretsIf, baseHRef, secure)
}

func newSso(
	ctx context.Context,
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
	clientSecretObj, err := secretsIf.Get(ctx, c.ClientSecret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	httpClientConfig := HTTPClientConfig{
		ClientCert:         c.ClientCert,
		ClientKey:          c.ClientKey,
		InsecureSkipVerify: c.InsecureSkipVerify,
		RootCA:             c.RootCA,
		RootCAFile:         c.RootCAFile,		
	}

	// Create http client
	httpClient, err := createHTTPClient(httpClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

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
	isSecretAlreadyExists := false
	if err != nil {
		isSecretAlreadyExists = apierr.IsAlreadyExists(err)
		if !isSecretAlreadyExists {
			return nil, fmt.Errorf("failed to create secret: %w", err)
		}
	}
	secret, err := secretsIf.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(secret.Data[cookieEncryptionPrivateKeySecretKey])
	if err != nil {
		if isSecretAlreadyExists {
			return nil, fmt.Errorf("failed to parse private key. If you have already defined a Secret named %s, delete it and retry: %w", secretName, err)
		} else {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
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

	var filterGroupsRegex []*regexp.Regexp
	if len(c.FilterGroupsRegex) > 0 {
		for _, regex := range c.FilterGroupsRegex {
			compiledRegex, err := regexp.Compile(regex)
			if err != nil {
				return nil, fmt.Errorf("failed to compile sso.filterGroupRegex: %s %w", regex, err)
			}
			filterGroupsRegex = append(filterGroupsRegex, compiledRegex)
		}
	}

	lf := logging.Fields{"redirectUrl": config.RedirectURL, "issuer": c.Issuer, "issuerAlias": "DISABLED", "clientId": c.ClientID, "scopes": config.Scopes, "insecureSkipVerify": c.InsecureSkipVerify, "filterGroupsRegex": c.FilterGroupsRegex}
	if c.IssuerAlias != "" {
		lf["issuerAlias"] = c.IssuerAlias
	}
	logger := logging.RequireLoggerFromContext(ctx).WithFields(lf)
	logger.Info(ctx, "SSO configuration")

	return &sso{
		config:            config,
		idTokenVerifier:   idTokenVerifier,
		baseHRef:          baseHRef,
		httpClient:        httpClient,
		secure:            secure,
		privateKey:        privateKey,
		encrypter:         encrypter,
		rbacConfig:        c.RBAC,
		expiry:            c.GetSessionExpiry(),
		customClaimName:   c.CustomGroupClaimName,
		userInfoPath:      c.UserInfoPath,
		issuer:            c.Issuer,
		filterGroupsRegex: filterGroupsRegex,
		logger:            logger,
	}, nil
}

func (s *sso) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	finalRedirectURL := r.URL.Query().Get("redirect")
	if !isValidFinalRedirectURL(finalRedirectURL) {
		finalRedirectURL = s.baseHRef
	}
	state, err := pkgrand.RandString(10)
	if err != nil {
		s.logger.WithError(err).Error(r.Context(), "failed to create state")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     state,
		Value:    finalRedirectURL,
		Expires:  time.Now().Add(3 * time.Minute),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})

	redirectOption := oauth2.SetAuthURLParam("redirect_uri", s.getRedirectURL(r))
	http.Redirect(w, r, s.config.AuthCodeURL(state, redirectOption), http.StatusFound)
}

func (s *sso) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(state)
	http.SetCookie(w, &http.Cookie{Name: state, MaxAge: 0})
	if err != nil {
		s.logger.WithError(err).Error(r.Context(), "failed to get cookie")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	redirectOption := oauth2.SetAuthURLParam("redirect_uri", s.getRedirectURL(r))
	// Use sso.httpClient in order to respect TLSOptions
	oauth2Context := context.WithValue(ctx, oauth2.HTTPClient, s.httpClient)
	oauth2Token, err := s.config.Exchange(oauth2Context, r.URL.Query().Get("code"), redirectOption)
	if err != nil {
		s.logger.WithError(err).Error(r.Context(), "failed to get oauth2Token by using code from the oauth2 server")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		s.logger.Error(r.Context(), "failed to extract id_token from the response")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	idToken, err := s.idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		s.logger.WithError(err).Error(r.Context(), "failed to verify the id token issued")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	c := &types.Claims{}
	if err := idToken.Claims(c); err != nil {
		s.logger.WithError(err).Error(r.Context(), "failed to get claims from the id token")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// Default to groups claim but if customClaimName is set
	// extract groups based on that claim key
	groups := c.Groups
	if s.customClaimName != "" {
		groups, err = c.GetCustomGroup(s.customClaimName)
		if err != nil {
			s.logger.Warn(r.Context(), err.Error())
		}
	}
	// Some SSO implementations (Okta) require a call to
	// the OIDC user info path to get attributes like groups
	if s.userInfoPath != "" {
		groups, err = c.GetUserInfoGroups(ctx, s.httpClient, oauth2Token.AccessToken, s.issuer, s.userInfoPath)
		if err != nil {
			s.logger.WithField("userInfoPath", s.userInfoPath).WithError(err).Error(r.Context(), "failed to get groups claim from the given userInfoPath")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	// only return groups that match at least one of the regexes
	if len(s.filterGroupsRegex) > 0 {
		var filteredGroups []string
		for _, group := range groups {
			for _, regex := range s.filterGroupsRegex {
				if regex.MatchString(group) {
					filteredGroups = append(filteredGroups, group)
					break
				}
			}
		}
		groups = filteredGroups
	}

	argoClaims := &types.Claims{
		Claims: jwt.Claims{
			Issuer:  issuer,
			Subject: c.Subject,
			Expiry:  jwt.NewNumericDate(time.Now().Add(s.expiry)),
		},
		Groups:                  groups,
		Email:                   c.Email,
		EmailVerified:           c.EmailVerified,
		Name:                    c.Name,
		ServiceAccountName:      c.ServiceAccountName,
		PreferredUsername:       c.PreferredUsername,
		ServiceAccountNamespace: c.ServiceAccountNamespace,
	}
	raw, err := jwt.Encrypted(s.encrypter).Claims(argoClaims).CompactSerialize()
	if err != nil {
		s.logger.WithError(err).Error(r.Context(), "failed to encrypt and serialize the jwt token")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	value := Prefix + raw
	s.logger.WithField("value", value).Debug(r.Context(), "handing oauth2 callback")
	http.SetCookie(w, &http.Cookie{
		Value:    value,
		Name:     "authorization",
		Path:     s.baseHRef,
		Expires:  time.Now().Add(s.expiry),
		SameSite: http.SameSiteStrictMode,
		Secure:   s.secure,
	})

	finalRedirectURL := cookie.Value
	if !isValidFinalRedirectURL(cookie.Value) {
		finalRedirectURL = s.baseHRef

	}
	http.Redirect(w, r, finalRedirectURL, http.StatusFound)
}

// isValidFinalRedirectURL checks whether the final redirect URL is safe.
//
// We only allow path-absolute-URL strings (e.g. /foo/bar), as defined in the
// WHATWG URL standard and RFC 3986:
// https://url.spec.whatwg.org/#path-absolute-url-string
// https://datatracker.ietf.org/doc/html/rfc3986#section-4.2
//
// It's not sufficient to only refer to RFC3986 for this validation logic
// because modern browsers will convert back slashes (\) to forward slashes (/)
// and will interprete percent-encoded bytes.
//
// We used to use absolute redirect URLs and would validate the scheme and host
// match the request scheme and host, but this led to problems when Argo is
// behind a TLS termination proxy, since the redirect URL would have the scheme
// "https" while the request scheme would be "http"
// (see https://github.com/argoproj/argo-workflows/issues/13031).
func isValidFinalRedirectURL(redirect string) bool {
	// Copied from https://github.com/oauth2-proxy/oauth2-proxy/blob/ab448cf38e7c1f0740b3cc2448284775e39d9661/pkg/app/redirect/validator.go#L47
	return strings.HasPrefix(redirect, "/") && !strings.HasPrefix(redirect, "//") && !invalidRedirectRegex.MatchString(redirect)
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

func (s *sso) getRedirectURL(r *http.Request) string {
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
