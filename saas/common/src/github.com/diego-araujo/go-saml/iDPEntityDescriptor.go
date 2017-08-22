package saml

import (
	"encoding/xml"
	"fmt"
)

func (s *ServiceProviderSettings) GetEntityDescriptor() (string, error) {
	d := EntityDescriptor{
		XMLName: xml.Name{
			Local: "md:EntityDescriptor",
		},
		DS:       "http://www.w3.org/2000/09/xmldsig#",
		XMLNS:    "urn:oasis:names:tc:SAML:2.0:metadata",
		MD:       "urn:oasis:names:tc:SAML:2.0:metadata",
		EntityId: s.Id,

		Extensions: Extensions{
			XMLName: xml.Name{
				Local: "md:Extensions",
			},
			Alg:    "urn:oasis:names:tc:SAML:metadata:algsupport",
			MDAttr: "urn:oasis:names:tc:SAML:metadata:attribute",
			MDRPI:  "urn:oasis:names:tc:SAML:metadata:rpi",

			UIInfo: UIInfo{
				XMLName: xml.Name{
					Local: "mdui:UIInfo",
				},
				MDUI: "urn:oasis:names:tc:SAML:metadata:ui",
				DisplayName: UIDisplayName{
					Lang:  "en",
					Value: "",
				},
				Description: UIDescription{
					Lang:  "en",
					Value: "",
				},
			},
		},
		SPSSODescriptor: SPSSODescriptor{
			ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
			AuthnRequestsSigned:        fmt.Sprintf("%t", s.SPSignRequest),
			WantAssertionsSigned:       fmt.Sprintf("%t", s.IDPSignResponse),

			Extensions: Extensions{
				XMLName: xml.Name{
					Local: "md:Extensions",
				},
				Alg:    "urn:oasis:names:tc:SAML:metadata:algsupport",
				MDAttr: "urn:oasis:names:tc:SAML:metadata:attribute",
				MDRPI:  "urn:oasis:names:tc:SAML:metadata:rpi",

				UIInfo: UIInfo{
					XMLName: xml.Name{
						Local: "mdui:UIInfo",
					},
					MDUI: "urn:oasis:names:tc:SAML:metadata:ui",
					DisplayName: UIDisplayName{
						Value: s.DisplayName,
						Lang:  "en",
					},
					Description: UIDescription{
						Lang:  "en",
						Value: s.Description,
					},
				},
			},
			SigningKeyDescriptor: KeyDescriptor{
				XMLName: xml.Name{
					Local: "md:KeyDescriptor",
				},

				Use: "signing",
				KeyInfo: KeyInfo{
					XMLName: xml.Name{
						Local: "ds:KeyInfo",
					},
					X509Data: X509Data{
						XMLName: xml.Name{
							Local: "ds:X509Data",
						},
						X509Certificate: X509Certificate{
							XMLName: xml.Name{
								Local: "ds:X509Certificate",
							},
							Cert: s.PublicCert(),
						},
					},
				},
			},
			EncryptionKeyDescriptor: KeyDescriptor{
				XMLName: xml.Name{
					Local: "md:KeyDescriptor",
				},

				Use: "encryption",
				KeyInfo: KeyInfo{
					XMLName: xml.Name{
						Local: "ds:KeyInfo",
					},
					X509Data: X509Data{
						XMLName: xml.Name{
							Local: "ds:X509Data",
						},
						X509Certificate: X509Certificate{
							XMLName: xml.Name{
								Local: "ds:X509Certificate",
							},
							Cert: s.PublicCert(),
						},
					},
				},
			},
			// SingleLogoutService{
			// 	XMLName: xml.Name{
			// 		Local: "md:SingleLogoutService",
			// 	},
			// 	Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect",
			// 	Location: "---TODO---",
			// },
			AssertionConsumerServices: []AssertionConsumerService{
				AssertionConsumerService{
					XMLName: xml.Name{
						Local: "md:AssertionConsumerService",
					},
					Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
					Location: s.AssertionConsumerServiceURL,
					Index:    "0",
					Default:  true,
				},
				//	AssertionConsumerService{
				//		XMLName: xml.Name{
				//				Local: "md:AssertionConsumerService",
				//		},
				//		Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact",
				//		Location: s.AssertionConsumerServiceURL,
				//		Index:    "1",
				//	},
			},
		},
	}
	b, err := xml.MarshalIndent(d, "", "    ")
	if err != nil {
		return "", err
	}

	newMetadata := fmt.Sprintf("<?xml version='1.0' encoding='UTF-8'?>\n%s", b)
	return string(newMetadata), nil
}
