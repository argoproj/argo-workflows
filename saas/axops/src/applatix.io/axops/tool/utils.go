// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/fixture"
	"applatix.io/axops/policy"
	"applatix.io/axops/project"
	"applatix.io/axops/service"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"fmt"
	"github.com/diego-araujo/go-saml"
	"os"
	"sync"
	"time"
)

var SCMRWMutex sync.RWMutex
var ActiveRepos map[string]string = map[string]string{}

const TmpPubCertFile = "pub_cert_test.crt"
const TmpPrvKeyFile = "prv_key_test.key"

var TestString = `<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:samlsig="http://www.w3.org/2000/09/xmldsig#" ID="_230981e8-4ae8-4467-4ff2-39225cc47b75" Destination="https://applatix.okta.com/app/applatix_applatix_1/exkmpf7ene19Ux7ml1t6/sso/saml" Version="2.0" ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" AssertionConsumerServiceURL="https://172.31.218.9:8443/v1/auth/saml/consume" IssueInstant="2016-09-07T01:48:22.91567983Z" AssertionConsumerServiceIndex="0" AttributeConsumingServiceIndex="0"><saml:Issuer xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">aws-vpc-b43e56d1-hong0-testvpc-160906-220649</saml:Issuer><samlp:NameIDPolicy AllowCreate="true" Format="urn:oasis:names:tc:SAML:2.0:nameid-format:transient"></samlp:NameIDPolicy><samlp:RequestedAuthnContext xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" Comparison="exact"><saml:AuthnContextClassRef xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport</saml:AuthnContextClassRef></samlp:RequestedAuthnContext><samlsig:Signature Id="Signature1" xmlns:dsig=""><samlsig:SignedInfo><samlsig:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"></samlsig:CanonicalizationMethod><samlsig:SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"></samlsig:SignatureMethod><samlsig:Reference URI="#_230981e8-4ae8-4467-4ff2-39225cc47b75"><samlsig:Transforms><samlsig:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"></samlsig:Transform></samlsig:Transforms><samlsig:DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"></samlsig:DigestMethod><samlsig:DigestValue></samlsig:DigestValue></samlsig:Reference></samlsig:SignedInfo><samlsig:SignatureValue></samlsig:SignatureValue><samlsig:KeyInfo><samlsig:X509Data><samlsig:X509Certificate>MIIDaDCCAlACCQCMRKLGFnfE2zANBgkqhkiG9w0BAQsFADB2MQswCQYDVQQGEwJVUzELMAkGA1UECBMCQ0ExEjAQBgNVBAcTCVN1bm55dmFsZTERMA8GA1UEChMIQXBwbGF0aXgxDjAMBgNVBAMTBWF4b3BzMSMwIQYJKoZIhvcNAQkBFhRzdXBwb3J0QGFwcGxhdGl4LmNvbTAeFw0xNjA4MDQxODE4MTRaFw0xNzA4MDQxODE4MTRaMHYxCzAJBgNVBAYTAlVTMQswCQYDVQQIEwJDQTESMBAGA1UEBxMJU3Vubnl2YWxlMREwDwYDVQQKEwhBcHBsYXRpeDEOMAwGA1UEAxMFYXhvcHMxIzAhBgkqhkiG9w0BCQEWFHN1cHBvcnRAYXBwbGF0aXguY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnaWSkDzYfBvp0CxHIEYVCnluKC75nzfY9QVqntgB04rrDfr+Yi6eHiXwWMbALURBp75hA9ZhOLxwRIt3LL9WRsWP+Seg68r0f5tfC7hT8kDy5cAK2sEW8YhbOKyP9dn8Jay9Mos9DqP+1SJoK4U9qm8s2Ee20VOqc9KJ0Fn5J3qSGPmkgNBqbeyRDSECXVLuvhO6Q5cSQinqc9O4cEJd1frKl1+6Wpkvn+1PjsIuFyuEiive16g0vJ+FZEjF4XWLVWUNeP2xr7JhhCF66lxVaO+lvuWavgoOP9AseRwvk4D+qMuIohPCjvLcZF1NMPOOCemdFq/WU4tyB25Y83Fs/wIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQBB0uZUQ7to+SowXUDeSNN1xSsdsCpVNzJ2sTVcabi2Aqm3lKj9Cf8QowuiKF6oif+TcK3VzTZr3q/jVJnENIKuSFoyVFDEAcPEX26ktz5Y7fepEGqv8LGPRxsAAKabaEfhPdOm1pF2laOkVyM9UCoFkgaIqeSww1bMtwF7zXcWEqaoMVdntTOu6HWGi8rTVEDqz31/BuyVHi1/Kg339kkaG6Y5DiPErVehO2iyI3JW5bsc8n/03dyL2hvUZ05DB0VMYuq92cHE/kxX5Fc432uww5LURbVcEx6gHefrlKle/HQg8OJmmTRwkkXQGuKvtv+9d05NCpBN1sMEdSWbsf5C</samlsig:X509Certificate></samlsig:X509Data></samlsig:KeyInfo></samlsig:Signature></samlp:AuthnRequest>`

var (
	ErrInvalidCertificate = axerror.ERR_API_INVALID_PARAM.NewWithMessage("The certificate is not valid.")
)

func ValidateCertKeyPair(publicCert, privateKey string) *axerror.AXError {
	if err := utils.WriteToFile(publicCert, TmpPubCertFile); err != nil {
		return err
	}
	defer os.Remove(TmpPubCertFile)

	if err := utils.WriteToFile(privateKey, TmpPrvKeyFile); err != nil {
		return err
	}
	defer os.Remove(TmpPrvKeyFile)

	signed, err := saml.SignRequest(TestString, TmpPrvKeyFile)
	if err != nil {
		fmt.Println(err)
		return ErrInvalidCertificate
	}

	fmt.Println(signed)

	err = saml.VerifyRequestSignature(signed, TmpPubCertFile)
	if err != nil {
		fmt.Println(err)
		return ErrInvalidCertificate
	}

	return nil
}

func AddActiveRepos(repos []string, id string) {
	SCMRWMutex.Lock()
	defer SCMRWMutex.Unlock()
	for _, repo := range repos {
		ActiveRepos[repo] = id
	}
}

func DeleteActiveRepos(repos []string) {
	SCMRWMutex.Lock()
	defer SCMRWMutex.Unlock()
	for _, repo := range repos {
		delete(ActiveRepos, repo)
	}
}

func DeleteDataByRepo(repo string) (*axerror.AXError, int) {
	SCMRWMutex.Lock()
	defer SCMRWMutex.Unlock()

	if err := policy.DeletePoliciesByRepo(repo, false); err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}

	if err := project.DeleteProjectsByRepo(repo, false); err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}

	if err := fixture.DeleteFixtureCategoryTemplatesByRepo(repo, false); err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}

	if err := service.DeleteTemplatesByRepo(repo, false); err != nil {
		return err, axerror.REST_INTERNAL_ERR
	}

	delete(ActiveRepos, repo)
	return nil, axerror.REST_STATUS_OK
}

func PurgeCachedBranchHeads(repo string) (*axerror.AXError, int) {
	params := map[string]interface{}{
		"repo": repo,
	}
	if axErr, code := utils.DevopsCl.Delete2("scm/branches", params, nil, nil); axErr != nil {
		utils.ErrorLog.Println("Delete cached branch heads for repo ", repo, " failed:", axErr)
		return axErr, code
	}
	return nil, 200
}

func AddExampleRepository() {
	admin, _ := user.GetUserByName("admin@internal")
	if admin == nil || time.Now().Unix()-admin.Ctime <= int64(time.Hour.Seconds())*24 {

		repos := []string{
			"https://github.com/argoproj/ci-workflow.git",
			"https://github.com/argoproj/appstore.git",
		}

		for _, repo := range repos {
			toolBase := &ToolBase{
				URL:      repo,
				Category: CategorySCM,
				Type:     TypeGIT,
			}
			gitHub := &GitHubConfig{}
			gitHub.ToolBase = toolBase
			example1 := &GitConfig{gitHub}
			axErr, _ := Create(example1)
			if axErr != nil {
				utils.ErrorLog.Printf("Failed to load the example repository(%v).\n", repo)
			} else {
				utils.InfoLog.Printf("Example repository(%v) is loaded to system.\n", repo)
			}
		}

	} else {
		utils.InfoLog.Println("Skip adding the example repositories, the cluster is up and running for more than 3 days.")
	}
}
