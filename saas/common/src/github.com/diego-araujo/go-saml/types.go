package saml

import "encoding/xml"

type AuthnRequest struct {
	XMLName                     xml.Name
	SAMLP                       string `xml:"xmlns:samlp,attr"`
	SAML                        string `xml:"xmlns:saml,attr"`
	SAMLSIG                     string `xml:"xmlns:samlsig,attr"`
	ID                          string `xml:"ID,attr"`
	Destination                 string `xml:"Destination,attr"`
	Version                     string `xml:"Version,attr"`
	ProtocolBinding             string `xml:"ProtocolBinding,attr"`
	AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr"`
	IssueInstant                string `xml:"IssueInstant,attr"`
	//AssertionConsumerServiceIndex  int                   `xml:"AssertionConsumerServiceIndex,attr"`
	//AttributeConsumingServiceIndex int                   `xml:"AttributeConsumingServiceIndex,attr"`
	Issuer       Issuer       `xml:"Issuer"`
	NameIDPolicy NameIDPolicy `xml:"NameIDPolicy"`
	//RequestedAuthnContext          RequestedAuthnContext `xml:"RequestedAuthnContext"`
	Signature      *Signature `xml:"Signature,omitempty"`
	originalString string
}

type Issuer struct {
	XMLName xml.Name
	SAML    string `xml:"xmlns:saml,attr"`
	Url     string `xml:",innerxml"`
}

type NameIDPolicy struct {
	XMLName     xml.Name
	AllowCreate bool   `xml:"AllowCreate,attr"`
	Format      string `xml:"Format,attr"`
}

type RequestedAuthnContext struct {
	XMLName              xml.Name
	SAMLP                string               `xml:"xmlns:samlp,attr"`
	Comparison           string               `xml:"Comparison,attr"`
	AuthnContextClassRef AuthnContextClassRef `xml:"AuthnContextClassRef"`
}

type AuthnContextClassRef struct {
	XMLName   xml.Name
	SAML      string `xml:"xmlns:saml,attr"`
	Transport string `xml:",innerxml"`
}

type Signature struct {
	XMLName        xml.Name
	Id             string `xml:"Id,attr"`
	SignedInfo     SignedInfo
	SignatureValue SignatureValue
	KeyInfo        KeyInfo
	DS             string `xml:"xmlns:dsig,attr"`
}

type SignedInfo struct {
	XMLName                xml.Name
	CanonicalizationMethod CanonicalizationMethod
	SignatureMethod        SignatureMethod
	SamlsigReference       SamlsigReference
}

type SignatureValue struct {
	XMLName xml.Name
	Value   string `xml:",innerxml"`
}

type KeyInfo struct {
	XMLName  xml.Name
	X509Data X509Data
}

type KeyInfoMain struct {
	XMLName      xml.Name     `xml:"KeyInfo"`
	EncryptedKey EncryptedKey `xml:"EncryptedKey,omitempty"`
}
type CanonicalizationMethod struct {
	XMLName   xml.Name
	Algorithm string `xml:"Algorithm,attr"`
}

type SignatureMethod struct {
	XMLName   xml.Name
	Algorithm string `xml:"Algorithm,attr"`
}

type SamlsigReference struct {
	XMLName      xml.Name
	URI          string       `xml:"URI,attr"`
	Transforms   Transforms   `xml:",innerxml"`
	DigestMethod DigestMethod `xml:",innerxml"`
	DigestValue  DigestValue  `xml:",innerxml"`
}

type X509Data struct {
	XMLName         xml.Name
	X509Certificate X509Certificate
}

type Transforms struct {
	XMLName   xml.Name
	Transform Transform
}

type DigestMethod struct {
	XMLName   xml.Name
	Algorithm string `xml:"Algorithm,attr"`
}

type DigestValue struct {
	XMLName xml.Name
}

type X509Certificate struct {
	XMLName xml.Name
	Cert    string `xml:",innerxml"`
}

type Transform struct {
	XMLName   xml.Name
	Algorithm string `xml:"Algorithm,attr"`
}

type EntityDescriptor struct {
	XMLName  xml.Name
	DS       string `xml:"xmlns:ds,attr"`
	XMLNS    string `xml:"xmlns,attr"`
	MD       string `xml:"xmlns:md,attr"`
	EntityId string `xml:"entityID,attr"`

	Extensions      Extensions      `xml:"Extensions"`
	SPSSODescriptor SPSSODescriptor `xml:"SPSSODescriptor"`
}

type Extensions struct {
	XMLName          xml.Name
	Alg              string `xml:"xmlns:alg,attr"`
	MDAttr           string `xml:"xmlns:mdattr,attr"`
	MDRPI            string `xml:"xmlns:mdrpi,attr"`
	EntityAttributes string `xml:"EntityAttributes,omitempty"`
	UIInfo           UIInfo
}

type UIInfo struct {
	XMLName     xml.Name
	DisplayName UIDisplayName
	MDUI        string `xml:"xmlns:mdui,attr"`
	Description UIDescription
}

type UIDisplayName struct {
	XMLName xml.Name `xml:"mdui:DisplayName"`
	Lang    string   `xml:"xml:lang,attr,omitempty"`
	Value   string   `xml:",innerxml"`
}

type UIDescription struct {
	XMLName xml.Name `xml:"mdui:Description"`
	Lang    string   `xml:"xml:lang,attr,omitempty"`
	Value   string   `xml:",innerxml"`
}

type SPSSODescriptor struct {
	XMLName                    xml.Name
	ProtocolSupportEnumeration string `xml:"protocolSupportEnumeration,attr"`
	AuthnRequestsSigned        string `xml:"AuthnRequestsSigned,attr"`
	WantAssertionsSigned       string `xml:"WantAssertionsSigned,attr"`
	SigningKeyDescriptor       KeyDescriptor
	EncryptionKeyDescriptor    KeyDescriptor
	// SingleLogoutService        SingleLogoutService `xml:"SingleLogoutService"`
	AssertionConsumerServices []AssertionConsumerService
	Extensions                Extensions `xml:"Extensions"`
}

type EntityAttributes struct {
	XMLName xml.Name
	SAML    string `xml:"xmlns:saml,attr"`

	EntityAttributes []Attribute `xml:"Attribute"` // should be array??
}

type SPSSODescriptors struct {
}

type KeyDescriptor struct {
	XMLName xml.Name
	Use     string  `xml:"use,attr"`
	KeyInfo KeyInfo `xml:"KeyInfo"`
}

type SingleLogoutService struct {
	Binding  string `xml:"Binding,attr"`
	Location string `xml:"Location,attr"`
}

type AssertionConsumerService struct {
	XMLName  xml.Name
	Binding  string `xml:"Binding,attr"`
	Location string `xml:"Location,attr"`
	Index    string `xml:"index,attr"`
	Default  bool   `xml:"isDefault,attr,omitempty"`
}

type Response struct {
	XMLName      xml.Name
	SAMLP        string `xml:"xmlns:saml2p,attr"`
	SAML         string `xml:"xmlns:saml,attr"`
	SAMLSIG      string `xml:"xmlns:samlsig,attr"`
	Destination  string `xml:"Destination,attr"`
	ID           string `xml:"ID,attr"`
	Version      string `xml:"Version,attr"`
	IssueInstant string `xml:"IssueInstant,attr"`
	InResponseTo string `xml:"InResponseTo,attr"`

	EncryptedAssertion EncryptedAssertion `xml:"EncryptedAssertion,omitempty"`
	Assertion          Assertion          `xml:"Assertion,omitempty"`
	Issuer             Issuer             `xml:"Issuer"`
	Status             Status             `xml:"Status"`
	Signature          Signature          `xml:"Signature,omitempty"`
	originalString     string
	decryptedString    string
}

type EncryptedAssertion struct {
	XMLName       xml.Name
	EncryptedData EncryptedData
	Assertion     Assertion `xml:"Assertion"`
}

type EncryptedData struct {
	XMLName          xml.Name
	EncryptionMethod EncryptionMethod
	KeyInfo          KeyInfoMain `xml:"KeyInfo"`
	CipherData       CipherData
}

type EncryptionMethod struct {
	XMLName      xml.Name
	Algorithm    string `xml:"Algorithm,attr"`
	DigestMethod DigestMethod
}

type EncryptedKey struct {
	XMLName          xml.Name
	ID               string `xml:"Id,attr"`
	EncryptionMethod EncryptionMethod
	KeyInfo          KeyInfo
	CipherData       CipherData
}

type CipherData struct {
	XMLName     xml.Name
	CipherValue string `xml:"CipherValue"`
}

type Assertion struct {
	XMLName            xml.Name
	ID                 string `xml:"ID,attr"`
	Version            string `xml:"Version,attr"`
	XS                 string `xml:"xmlns:xs,attr"`
	XSI                string `xml:"xmlns:xsi,attr"`
	SAML               string `xml:"saml,attr"`
	IssueInstant       string `xml:"IssueInstant,attr"`
	Issuer             Issuer `xml:"Issuer"`
	Subject            Subject
	Conditions         Conditions
	AttributeStatement AttributeStatement
	Signature          Signature
}

type Conditions struct {
	XMLName      xml.Name
	NotBefore    string `xml:",attr"`
	NotOnOrAfter string `xml:",attr"`
}

type Subject struct {
	XMLName             xml.Name
	NameID              NameID
	SubjectConfirmation SubjectConfirmation
}

type SubjectConfirmation struct {
	XMLName                 xml.Name
	Method                  string `xml:",attr"`
	SubjectConfirmationData SubjectConfirmationData
}

type Status struct {
	XMLName    xml.Name
	StatusCode StatusCode `xml:"StatusCode"`
}

type SubjectConfirmationData struct {
	InResponseTo string `xml:",attr"`
	NotOnOrAfter string `xml:",attr"`
	Recipient    string `xml:",attr"`
}

type NameID struct {
	XMLName xml.Name
	Format  string `xml:",attr"`
	Value   string `xml:",innerxml"`
}

type StatusCode struct {
	XMLName xml.Name
	Value   string `xml:",attr"`
}

type AttributeValue struct {
	XMLName xml.Name
	Type    string `xml:"xsi:type,attr"`
	Value   string `xml:",innerxml"`
}

type Attribute struct {
	XMLName        xml.Name
	Name           string `xml:",attr"`
	FriendlyName   string `xml:",attr"`
	NameFormat     string `xml:",attr"`
	AttributeValue AttributeValue
}

type AttributeStatement struct {
	XMLName    xml.Name
	Attributes []Attribute `xml:"Attribute"`
}
