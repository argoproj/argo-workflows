package sso

import (
	"fmt"
	"os"
	"net/http"
	"testing"
)

// Create a temporary PEM file with the given content
func createTempPemFile(content string) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "*.pem")
	if err != nil {
		return "", nil, fmt.Errorf("unable to create temp file: %w", err)
	}

	_, err = tmpFile.Write([]byte(content))
	if err != nil {
		tmpFile.Close()
		return "", nil, fmt.Errorf("unable to write to temp file: %w", err)
	}

	err = tmpFile.Close()
	if err != nil {
		return "", nil, fmt.Errorf("unable to close temp file: %w", err)
	}

	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }, nil
}

// Mock for successful certificate loading
func TestCreateHttpClient_Success(t *testing.T) {

	certFilePath, cleanupCert, err := createTempPemFile(certContent)
	if err != nil {
		t.Fatalf("unable to create temporary cert file: %v", err)
	}
	defer cleanupCert()

	keyFilePath, cleanupKey, err := createTempPemFile(keyContent)
	if err != nil {
		t.Fatalf("unable to create temporary key file: %v", err)
	}
	defer cleanupKey()

	config := HTTPClientConfig{
		ClientCert:         certFilePath,
		ClientKey:          keyFilePath,
		InsecureSkipVerify: false,
		RootCA:             certContent,
		RootCAFile:         certFilePath,
	}

	httpClient, err := createHTTPClient(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if httpClient == nil {
		t.Fatal("expected non-nil httpClient")
	}

	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected httpClient.Transport to be of type *http.Transport")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be set")
	}

	if transport.TLSClientConfig.InsecureSkipVerify != false {
		t.Errorf("expected InsecureSkipVerify to be false, got %v", transport.TLSClientConfig.InsecureSkipVerify)
	}

	if len(transport.TLSClientConfig.Certificates) == 0 {
		t.Fatal("expected Certificates to be set")
	}

	if transport.TLSClientConfig.RootCAs == nil {
		t.Fatal("expected Root CA to be set")
	}
}

// Mock for certificate loading failure
func TestCreateHttpClient_LoadCertError(t *testing.T) {
	// Provide invalid paths for certificates to simulate an error
	config := HTTPClientConfig{
		ClientCert:         "invalid_cert.pem",
		ClientKey:          "invalid_key.pem",
		InsecureSkipVerify: false,
	}

	_, err := createHTTPClient(config)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "unable to load client certificate: open invalid_cert.pem: no such file or directory" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Mock for using default settings
func TestCreateHttpClient_DefaultSettings(t *testing.T) {
	config := HTTPClientConfig{
		ClientCert:         "",
		ClientKey:          "",
		InsecureSkipVerify: false,
	}

	httpClient, err := createHTTPClient(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if httpClient == nil {
		t.Fatal("expected non-nil httpClient")
	}

	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected httpClient.Transport to be of type *http.Transport")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be set")
	}

	if transport.TLSClientConfig.InsecureSkipVerify != false {
		t.Errorf("expected InsecureSkipVerify to be false, got %v", transport.TLSClientConfig.InsecureSkipVerify)
	}

	if len(transport.TLSClientConfig.Certificates) != 0 {
		t.Fatal("expected no Certificates to be set")
	}

	if transport.TLSClientConfig.RootCAs != nil {
		t.Fatal("expected no certificate authorities to be set")
	}
}

const (
	certContent = `-----BEGIN CERTIFICATE-----
MIIDZTCCAk2gAwIBAgIUSCZBzVxtJXm7TOiySB2puPjMRHkwDQYJKoZIhvcNAQEL
BQAwQjELMAkGA1UEBhMCVVMxETAPBgNVBAgMCENvbG9yYWRvMSAwHgYDVQQKDBdG
YWtlIENlcnRzIEluY29ycG9yYXRlZDAeFw0yNDA5MDEwNzIzMThaFw0yNTA5MDEw
NzIzMThaMEIxCzAJBgNVBAYTAlVTMREwDwYDVQQIDAhDb2xvcmFkbzEgMB4GA1UE
CgwXRmFrZSBDZXJ0cyBJbmNvcnBvcmF0ZWQwggEiMA0GCSqGSIb3DQEBAQUAA4IB
DwAwggEKAoIBAQC3dRw+Um2ZI0Nam75nxpu6Plz9IlHmNnt8kepkIQvKoJP7hd8R
2qFgXELvQWQf8p7+SGczHu11Fk+Br+4VSgt3Hv67v7EJhDIz5mdpnoh4yowWcHCJ
5/2fV5ZicvWcmMFXI3y6c5UiXF+mmVQPXa86jQQWCjkFi7n1zQ2901d0h5OwNL7j
lw2/YfYKLqdrJACsXw1ay4cIDq6uXia64OPVNKIb4b/22VlDnpKkQg0r3dj1q4yw
+hiDAE9/CBspt7cxmvM7bU75yM42sOyi4G1b0qNE2jpIWZ2jgPTqb2CoV9tNJZJo
UtgHoEl2aLPV7e/nF15bgmA3bfsCbfo05DJfAgMBAAGjUzBRMB0GA1UdDgQWBBR/
Lk7H386KYfk1BJD7gwtQgIF3IjAfBgNVHSMEGDAWgBR/Lk7H386KYfk1BJD7gwtQ
gIF3IjAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQB2c9XIZVxa
n9gdNCUiMCmnRPwUCIF6ZSIFnbMOR0xee7I9rQyqX1jFTDmxwZo1JSPi/jDWkeRg
11pP0zxD8itCv0+3MRKdG52zqXxYAk90qiPn2/pz6OeMFmcxVUz6NdjLk95Gh1vo
aJohLXkvstxU8BHVCpsgK22zFBO/v+HhLvc15d1roTddoY8oUA6qJXTbnxeekfKV
QkvB2HoREJBm2SLClYq4v8IiE/ezpXNmXT18KT5v52biav9BDhNCZUeXBezgLOkS
Wh0/k3cgHK6tmKF1AhByLRXkPmD+O9RCDBxZxqRsydHvWETL8gXLQb3fMEhxvseC
oFPJ0Zcg2K8U
-----END CERTIFICATE-----
`

	keyContent = `-----BEGIN RSA PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC3dRw+Um2ZI0Na
m75nxpu6Plz9IlHmNnt8kepkIQvKoJP7hd8R2qFgXELvQWQf8p7+SGczHu11Fk+B
r+4VSgt3Hv67v7EJhDIz5mdpnoh4yowWcHCJ5/2fV5ZicvWcmMFXI3y6c5UiXF+m
mVQPXa86jQQWCjkFi7n1zQ2901d0h5OwNL7jlw2/YfYKLqdrJACsXw1ay4cIDq6u
Xia64OPVNKIb4b/22VlDnpKkQg0r3dj1q4yw+hiDAE9/CBspt7cxmvM7bU75yM42
sOyi4G1b0qNE2jpIWZ2jgPTqb2CoV9tNJZJoUtgHoEl2aLPV7e/nF15bgmA3bfsC
bfo05DJfAgMBAAECggEAFfAmMXmv63khC8vGCCji5HGisw6QlqP7PllAmzqsa02q
hJBsrXjkhV5jDrNWIs/jnWrRFHblVHQXi92a7ebN2i/VrGPu6sFpM3Wg9itkDHXE
LMbDXmpklNJnhFxU7KYDsMTonG9H7TT4pzZ8q927H5hPXcdZLEWaNj+QHhwQwDlm
HfdRJlU0uY/iTZVVMo6jxm1E3xN2EoNaFlz3LkxthC204gVrjLJHB3LUQ64x3C2r
ZwYeKjEsulPqEEMjWJ4mzddX9yKXVqg5AMmlLIDh1M+9v6mkpKSnrCDQAenyF1Tk
26pxkO3iDKBhU03SCOjYEiglQrSWbQ8e440R/XsDcQKBgQDwh33GnGM7xX8cpj61
2CidMawJk9uVypKIiC1vXrNN11vp0LdaTDmXf/M23vxTMwVhVhO+6BEzVFM4cQ2v
pOWAtFbcwEMR98NL8LuUdMcWR1MWUAQ91GF2TZcNuz94cQ/3uKVghUF5p7dT0kWJ
bn1pZWune6sV/9NYeYxOmdCJiQKBgQDDQeK851FAAFWCfBbKR/Rayb+dKc4YkgJz
P8xh7lYpnr/1F9aPBG1lQ7C5miOKZhg2I5PYW7r4VzFwKRWDNZdFOMLvO/BfRBXr
CAk01MHAuwIxzmiL6Snkm2CjUHE0lDb+pDxXRuEOlrVykIiEDBPzrL/jVkqP+D9m
1TCKPHoqpwKBgQDnDZCx+EqPAVHwyHXXIvUow616adF3G+gVRZM3t6XQcb82ZSus
jyqHsP6GyD9lAM77SL+hFLZpM2jaACfggSuBrjr+xaXoHbQ6P99BZchVS2CyP11D
s7+H8FLZevUmkp1/Hp2mkXtrDMRbvdLUiRHp6+Y1NeQMNvrjs6cnXjRn2QKBgGZ6
QeIbFY2dn0NolR19PkYX9LUrp7tFhnuuVDphuF8Hrn+YD0fobvHi4PHIcDbG9pYT
fhjjq/GC8bOIHH5MtiPicozUzIdzWH2OLibIMxhQDgrN5hjoOtB8q++K3J9X2rUy
xWiZDq11c625Ja0IGcCePee29lMxWzVBVsR2kTepAoGAcPA/apBJhgFpCZeF9UKX
J3fSlNcYYNL19/6svsKdqHEEbFwiH8YtRVvqtarFWokYUWa/f98WdkD3Ltfew8Mm
fEAiZ/6dZzWVJS/XqEQ40ThHBkK2L8kW8Sg2We8IXRRe2Ao4nt9ErE6bbl75phUP
jtX0dItI5GInxIe+bG5qPaM=
-----END RSA PRIVATE KEY-----
`
)
