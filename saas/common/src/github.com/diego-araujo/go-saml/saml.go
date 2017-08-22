package saml

import "github.com/diego-araujo/go-saml/util"
import stderrors "errors"

// ServiceProviderSettings provides settings to configure server acting as a SAML Service Provider.
// Expect only one IDP per SP in this configuration. If you need to configure multipe IDPs for an SP
// then configure multiple instances of this module
type ServiceProviderSettings struct {
	PublicCertPath              string
	PrivateKeyPath              string
	IDPSSOURL                   string
	IDPSSODescriptorURL         string
	DisplayName                 string
	Description                 string
	IDPPublicCertPath           string
	AssertionConsumerServiceURL string
	Id                          string
	SPSignRequest               bool
	IDPSignResponse             bool
	IDPSignResponseAssertion    bool
	hasInit                     bool
	publicCert                  string
	privateKey                  string
	iDPPublicCert               string
}

var (
	ErrPrivkey    = stderrors.New("error load private key")
	ErrSpPubCert  = stderrors.New("error load SP publicCert")
	ErrIdpPubCert = stderrors.New("error load IDP publicCert")
)

type IdentityProviderSettings struct {
}

func (s *ServiceProviderSettings) Init() (err error) {
	if s.hasInit {
		return nil
	}
	s.hasInit = true

	if s.SPSignRequest {
		s.publicCert, err = util.LoadCertificate(s.PublicCertPath)
		if err != nil {
			return ErrSpPubCert
		}

		s.privateKey, err = util.LoadCertificate(s.PrivateKeyPath)
		if err != nil {
			return ErrPrivkey
		}
	}

	if s.IDPSignResponse || s.IDPSignResponseAssertion {
		s.iDPPublicCert, err = util.LoadCertificate(s.IDPPublicCertPath)
		if err != nil {
			return ErrIdpPubCert
		}
	}

	return nil
}

func (s *ServiceProviderSettings) PublicCert() string {
	return s.publicCert
}

func (s *ServiceProviderSettings) PrivateKey() string {
	return s.privateKey
}

func (s *ServiceProviderSettings) IDPPublicCert() string {
	return s.iDPPublicCert
}
