package saml

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/diego-araujo/go-saml/util"
	"regexp"
	"strings"
	"time"
)

func ParseCompressedEncodedResponse(b64ResponseXML string) (*Response, error) {
	authnResponse := Response{}
	compressedXML, err := base64.StdEncoding.DecodeString(b64ResponseXML)
	if err != nil {
		return nil, err
	}
	bXML := util.Decompress(compressedXML)
	err = xml.Unmarshal(bXML, &authnResponse)
	if err != nil {
		return nil, err
	}

	// There is a bug with XML namespaces in Go that's causing XML attributes with colons to not be roundtrip
	// marshal and unmarshaled so we'll keep the original string around for validation.
	authnResponse.originalString = string(bXML)
	return &authnResponse, nil

}

func ParseEncodedResponse(b64ResponseXML string) (*Response, error) {
	response := Response{}
	bytesXML, err := base64.StdEncoding.DecodeString(b64ResponseXML)
	//dst := string(bytesXML[:])
	if err != nil {
		return nil, err
	}
	err = xml.Unmarshal(bytesXML, &response)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("%+v\n", response)
	// There is a bug with XML namespaces in Go that's causing XML attributes with colons to not be roundtrip
	// marshal and unmarshaled so we'll keep the original string around for validation.
	response.originalString = string(bytesXML)
	return &response, nil
}

func (r *Response) IsEncrypted() bool {

	//Test if exits EncryptedAssertion tag
	if r.EncryptedAssertion.EncryptedData.EncryptionMethod.Algorithm == "" {
		return false
	} else {
		return true
	}
}

func (r *Response) Decrypt(privateKeyPath string) error {
	s := r.originalString

	if r.IsEncrypted() == false {
		return errors.New("missing EncryptedAssertion tag on SAML Response, is encrypted?")

	}
	plainXML, err := DecryptResponse(s, privateKeyPath)
	if err != nil {
		if strings.Contains(s, "RetrievalMethod") {
			plainXML, err = r.DecryptAdvanced(privateKeyPath)
			if err != nil {
				return err
			}
		}
	}
	err = xml.Unmarshal([]byte(plainXML), &r)
	if err != nil {
		return err
	}

	r.decryptedString = plainXML
	return nil
}

func (r *Response) DecryptAdvanced(privateKeyPath string) (string, error) {
	cp := r.originalString

	// find the EncryptedKey namespace string
	re := regexp.MustCompile(`(?s).*EncryptedKey\s+xmlns:(\w{1,10})="http://www.w3.org/2001/04/xmlenc#.*"`)
	matches := re.FindStringSubmatch(cp)
	if len(matches) < 2 {
		return "", errors.New("The namespace <http://www.w3.org/2001/04/xmlenc#> for EncryptedKey attribute is missing. Pleace check your SAML response format.")
	}
	xmlenc := matches[1]
	fmt.Println("xmlenc", xmlenc)

	// find the EncryptedKey body
	keyPtn := fmt.Sprintf(`(?s).*EncryptedAssertion.*(<%s:EncryptedKey\s+xmlns:%s=.*</%s:EncryptedKey>).*EncryptedAssertion>.*`, xmlenc, xmlenc, xmlenc)
	re = regexp.MustCompile(keyPtn)
	matches = re.FindStringSubmatch(cp)
	if len(matches) < 2 {
		return "", errors.New("The EncryptedKey attribute is missing. Pleace check your SAML response format.")
	}
	keyBody := matches[1]
	fmt.Println("keyBody", keyBody)

	// find the KeyInfo namspace string
	re = regexp.MustCompile(`(?s).*EncryptedData.*KeyInfo\s+xmlns:(\w{1,10})="http://www.w3.org/2000/09/xmldsig#".*KeyInfo>.*EncryptedData>.*`)
	matches = re.FindStringSubmatch(cp)
	if len(matches) < 2 {
		return "", errors.New("The namespace <http://www.w3.org/2000/09/xmldsig#> for KeyInfo attribute is missing. Pleace check your SAML response format.")
	}
	ds := matches[1]
	fmt.Println("ds", ds)

	// find the RetrievalMethod body
	rtrvPtn := fmt.Sprintf(`(?s).*KeyInfo.*(<%s:RetrievalMethod\s+Type="http://www.w3.org/2001/04/xmlenc#EncryptedKey"\s+URI="#\w{1,50}"/>).*KeyInfo>.*"`, ds)
	re = regexp.MustCompile(rtrvPtn)
	matches = re.FindStringSubmatch(cp)
	if len(matches) < 2 {
		return "", errors.New("The RetrievalMethod attribute is missing. Pleace check your SAML response format.")
	}
	rtrvBody := matches[1]
	fmt.Println("rtrvBody", rtrvBody)

	// convert the response body
	cp = strings.Replace(cp, keyBody, "", -1)
	cp = strings.Replace(cp, rtrvBody, rtrvBody+keyBody, -1)
	fmt.Println("ConvertedResponse", cp)

	return DecryptResponse(cp, privateKeyPath)
}

func (r *Response) ValidateAssertionSignature(s *ServiceProviderSettings) error {

	assertion, err := r.getAssertion()
	if err != nil {
		return err
	}

	responseStr := r.originalString
	if r.IsEncrypted() {
		responseStr = r.decryptedString
	}

	if len(assertion.Signature.SignatureValue.Value) == 0 {
		return errors.New("There is no signature for SAML Response assertion. Please check your IdP configuration.")
	}

	if len(r.Signature.SignatureValue.Value) == 0 {
		err = VerifyAssertionSignature(responseStr, s.IDPPublicCertPath)
		if err != nil {
			return err
		}
	} else {
		cp := responseStr
		re := regexp.MustCompile(`(?s).*Signature\s+xmlns:(\w{1,10})="http://www.w3.org/2000/09/xmldsig#".*`)
		matches := re.FindStringSubmatch(cp)
		if len(matches) < 2 {
			return errors.New("The namespace <http://www.w3.org/2000/09/xmldsig#> for Signature attribute is missing. Pleace check your SAML response format.")
		}
		ns := matches[1]

		sigPtn := fmt.Sprintf(`(<%s:Signature\s+xmlns:%s=.*</%s:Signature>)`, ns, ns, ns)
		re = regexp.MustCompile(fmt.Sprintf(`^(?s).*%s.*%s.*`, sigPtn, sigPtn))
		matches = re.FindStringSubmatch(cp)
		if len(matches) != 3 {
			return errors.New("There should be signatures for both SAML reponse and asssertion. Please check your IdP configuration.")
		} else {
			signatures := matches[1:3]
			for _, signature := range signatures {
				fmt.Println(signature)
				if strings.Contains(signature, r.Signature.SignatureValue.Value) {
					cp = strings.Replace(cp, signature, "", -1)
				}
			}
			fmt.Println(cp)
			err = VerifyAssertionSignature(cp, s.IDPPublicCertPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Response) ValidateResponseSignature(s *ServiceProviderSettings) error {

	if len(r.Signature.SignatureValue.Value) == 0 {
		return errors.New("There is no signature for SAML Response. Please check your IdP configuration.")
	}

	err := VerifyResponseSignature(r.originalString, s.IDPPublicCertPath)
	if err != nil {
		return err
	}

	return nil
}

func (r *Response) getAssertion() (Assertion, error) {

	assertion := Assertion{}

	if r.IsEncrypted() {
		assertion = r.EncryptedAssertion.Assertion
	} else {
		assertion = r.Assertion
	}

	if len(assertion.ID) == 0 {
		return assertion, errors.New("Therer is no assertion. Please check your IdP configuration.")
	}
	return assertion, nil
}

func (r *Response) Validate(s *ServiceProviderSettings) error {
	if r.Version != "2.0" {
		return errors.New("Only SAML 2.0 is supported. Please check your SAML version.")
	}

	if len(r.ID) == 0 {
		return errors.New("ID attritue is missing in SAML response. Please check your IdP configuration.")
	}

	assertion, err := r.getAssertion()
	if err != nil {
		return err
	}

	if assertion.Subject.SubjectConfirmation.Method != "urn:oasis:names:tc:SAML:2.0:cm:bearer" {
		return errors.New("SAML assertion method is not supported.")
	}

	if assertion.Subject.SubjectConfirmation.SubjectConfirmationData.Recipient != s.AssertionConsumerServiceURL {
		return errors.New("SAML response subject recipient mismatch, expected: " + s.AssertionConsumerServiceURL + " not " + assertion.Subject.SubjectConfirmation.SubjectConfirmationData.Recipient)
	}

	if r.Destination != s.AssertionConsumerServiceURL {
		return errors.New("SAML response destination mismatch, expected: " + s.AssertionConsumerServiceURL + " not " + r.Destination)
	}

	return nil
}

func (r *Response) ValidateExpiredConfirmation(s *ServiceProviderSettings) error {

	assertion, err := r.getAssertion()
	if err != nil {
		return err
	}

	//CHECK TIMES
	expires := assertion.Subject.SubjectConfirmation.SubjectConfirmationData.NotOnOrAfter
	notOnOrAfter, e := time.Parse(time.RFC3339, expires)
	if e != nil {
		return e
	}
	if notOnOrAfter.Before(time.Now()) {
		return errors.New("assertion has expired on: " + expires)
	}

	return nil
}
func NewSignedResponse() *Response {
	return &Response{
		XMLName: xml.Name{
			Local: "samlp:Response",
		},
		SAMLP:        "urn:oasis:names:tc:SAML:2.0:protocol",
		SAML:         "urn:oasis:names:tc:SAML:2.0:assertion",
		SAMLSIG:      "http://www.w3.org/2000/09/xmldsig#",
		ID:           util.ID(),
		Version:      "2.0",
		IssueInstant: time.Now().UTC().Format(time.RFC3339Nano),
		Issuer: Issuer{
			XMLName: xml.Name{
				Local: "saml:Issuer",
			},
			Url: "", // caller must populate ar.AppSettings.AssertionConsumerServiceURL,
		},
		Signature: Signature{
			XMLName: xml.Name{
				Local: "samlsig:Signature",
			},
			Id: "Signature1",
			SignedInfo: SignedInfo{
				XMLName: xml.Name{
					Local: "samlsig:SignedInfo",
				},
				CanonicalizationMethod: CanonicalizationMethod{
					XMLName: xml.Name{
						Local: "samlsig:CanonicalizationMethod",
					},
					Algorithm: "http://www.w3.org/2001/10/xml-exc-c14n#",
				},
				SignatureMethod: SignatureMethod{
					XMLName: xml.Name{
						Local: "samlsig:SignatureMethod",
					},
					Algorithm: "http://www.w3.org/2000/09/xmldsig#rsa-sha1",
				},
				SamlsigReference: SamlsigReference{
					XMLName: xml.Name{
						Local: "samlsig:Reference",
					},
					URI: "", // caller must populate "#" + ar.Id,
					Transforms: Transforms{
						XMLName: xml.Name{
							Local: "samlsig:Transforms",
						},
						Transform: Transform{
							XMLName: xml.Name{
								Local: "samlsig:Transform",
							},
							Algorithm: "http://www.w3.org/2000/09/xmldsig#enveloped-signature",
						},
					},
					DigestMethod: DigestMethod{
						XMLName: xml.Name{
							Local: "samlsig:DigestMethod",
						},
						Algorithm: "http://www.w3.org/2000/09/xmldsig#sha1",
					},
					DigestValue: DigestValue{
						XMLName: xml.Name{
							Local: "samlsig:DigestValue",
						},
					},
				},
			},
			SignatureValue: SignatureValue{
				XMLName: xml.Name{
					Local: "samlsig:SignatureValue",
				},
			},
			KeyInfo: KeyInfo{
				XMLName: xml.Name{
					Local: "samlsig:KeyInfo",
				},
				X509Data: X509Data{
					XMLName: xml.Name{
						Local: "samlsig:X509Data",
					},
					X509Certificate: X509Certificate{
						XMLName: xml.Name{
							Local: "samlsig:X509Certificate",
						},
						Cert: "", // caller must populate cert,
					},
				},
			},
		},
		Status: Status{
			XMLName: xml.Name{
				Local: "samlp:Status",
			},
			StatusCode: StatusCode{
				XMLName: xml.Name{
					Local: "samlp:StatusCode",
				},
				// TODO unsuccesful responses??
				Value: "urn:oasis:names:tc:SAML:2.0:status:Success",
			},
		},
		Assertion: Assertion{
			XMLName: xml.Name{
				Local: "saml:Assertion",
			},
			XS:           "http://www.w3.org/2001/XMLSchema",
			XSI:          "http://www.w3.org/2001/XMLSchema-instance",
			SAML:         "urn:oasis:names:tc:SAML:2.0:assertion",
			Version:      "2.0",
			ID:           util.ID(),
			IssueInstant: time.Now().UTC().Format(time.RFC3339Nano),
			Issuer: Issuer{
				XMLName: xml.Name{
					Local: "saml:Issuer",
				},
				Url: "", // caller must populate ar.AppSettings.AssertionConsumerServiceURL,
			},
			Subject: Subject{
				XMLName: xml.Name{
					Local: "saml:Subject",
				},
				NameID: NameID{
					XMLName: xml.Name{
						Local: "saml:NameID",
					},
					Format: "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified",
					Value:  "",
				},
				SubjectConfirmation: SubjectConfirmation{
					XMLName: xml.Name{
						Local: "saml:SubjectConfirmation",
					},
					Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer",
					SubjectConfirmationData: SubjectConfirmationData{
						InResponseTo: "",
						NotOnOrAfter: time.Now().Add(time.Minute * 5).UTC().Format(time.RFC3339Nano),
						Recipient:    "",
					},
				},
			},
			Conditions: Conditions{
				XMLName: xml.Name{
					Local: "saml:Conditions",
				},
				NotBefore:    time.Now().Add(time.Minute * -5).UTC().Format(time.RFC3339Nano),
				NotOnOrAfter: time.Now().Add(time.Minute * 5).UTC().Format(time.RFC3339Nano),
			},
			AttributeStatement: AttributeStatement{
				XMLName: xml.Name{
					Local: "saml:AttributeStatement",
				},
				Attributes: []Attribute{},
			},
		},
	}
}

// AddAttribute add strong attribute to the Response
func (r *Response) AddAttribute(name, value string) {
	r.Assertion.AttributeStatement.Attributes = append(r.Assertion.AttributeStatement.Attributes, Attribute{
		XMLName: xml.Name{
			Local: "saml:Attribute",
		},
		Name:       name,
		NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
		AttributeValue: AttributeValue{
			XMLName: xml.Name{
				Local: "saml:AttributeValue",
			},
			Type:  "xs:string",
			Value: value,
		},
	})
}

func (r *Response) String() (string, error) {
	b, err := xml.MarshalIndent(r, "", "    ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (r *Response) OriginalString() string {
	return r.originalString
}

func (r *Response) SignedString(privateKeyPath string) (string, error) {
	s, err := r.String()
	if err != nil {
		return "", err
	}

	return SignResponse(s, privateKeyPath)
}

func (r *Response) EncodedSignedString(privateKeyPath string) (string, error) {
	signed, err := r.SignedString(privateKeyPath)
	if err != nil {
		return "", err
	}
	b64XML := base64.StdEncoding.EncodeToString([]byte(signed))
	return b64XML, nil
}

func (r *Response) CompressedEncodedSignedString(privateKeyPath string) (string, error) {
	signed, err := r.SignedString(privateKeyPath)
	if err != nil {
		return "", err
	}
	compressed := util.Compress([]byte(signed))
	b64XML := base64.StdEncoding.EncodeToString(compressed)
	return b64XML, nil
}

// GetAttribute by Name or by FriendlyName. Return blank string if not found
func (r *Response) GetAttribute(name string) string {
	attrStatement := AttributeStatement{}

	if r.IsEncrypted() {
		attrStatement = r.EncryptedAssertion.Assertion.AttributeStatement
	} else {
		attrStatement = r.Assertion.AttributeStatement
	}

	for _, attr := range attrStatement.Attributes {
		if attr.Name == name || attr.FriendlyName == name {
			return attr.AttributeValue.Value
		}
	}
	return ""
}

func (r *Response) GetNameID() string {
	if r.IsEncrypted() {
		return r.EncryptedAssertion.Assertion.Subject.NameID.Value
	} else {
		return r.Assertion.Subject.NameID.Value
	}
}
