package commands

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

const argoCert = `
-----BEGIN CERTIFICATE-----
MIIEyjCCArICCQDIyfmM271aGjANBgkqhkiG9w0BAQsFADAnMRIwEAYDVQQDDAls
b2NhbGhvc3QxETAPBgNVBAoMCEFyZ29Qcm9qMB4XDTIxMDMyNjE0MjAwNFoXDTIy
MDMyNjE0MjAwNFowJzESMBAGA1UEAwwJbG9jYWxob3N0MREwDwYDVQQKDAhBcmdv
UHJvajCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAPLEgWjhqFUDJoTU
5OqqPhGQmi87IaD/VxghxQeKt9cBiD3ePWGGBAy279L5wLrrJPM1bVT9k0jJ7T2e
EaC4ibayXxKpfLiR90EypS/OnotJIX5ZrU++IVzQRZ40DnQrkjwSz31adSDIM3gi
NJuR7KmsOTIWFy9nUq47+/ZgTlen0+sH+23jvxpKCX8+i/xfVNT28TABcSCWx3c+
uzhKQgmuN5VeY7GKzNaFsK6TzKAy3Wv+f6Za+b5vmMkXDmjVeAazkj9JeSnNLNd3
3ocwipaa0JFdGTRuZZkRaMh3iUcfxJqQwxO4qy59KMD2LpwUreL/N8V9JjPilXRu
Z8Ld8NpgH7GoG5hEW9r13Qcul5etYKa2Ux+hUm2rBdp0MEx6FOzCx1568wjWYCx2
fp5UT/S8GyoTgL9zF8wyLFZo/B+772+TFVojhqjWwRxdQN18suMc/tUnXLJIyUIp
E9XASnjfvMVMxLTsTAjbBx05K/cEbXEpyfeRwzKBR1a5PcJcFF7mORHmqV3bZOhN
RwTGoZub3rns4XXrJEHKSa07JmOazkIwu+9J4RzLYwDs5k2ZnNJA31t322mCpQ3d
rsSE7E96CTtIPySRtVdO+a19H5K9IaPSe/fAspTxjcHbTeHJUPh+z4fCoXW5rxyM
WkpV9bEc1RLtCtIJy8U5kEgKwJgPAgMBAAEwDQYJKoZIhvcNAQELBQADggIBAGSs
Oa2vMCPa3EdUvh630Ng3c7jwjQbR+CeeMr0ybeRuxuVevXCOOk1pH/d9IdV9Ybfb
rSXYqh3AjyLN8gBxhCj0hEIxL7GKX1jBRaAIewAjjcGHb2ZWZu7HTskmmx/6CXVr
At/flQ4iHY7ejDGeutpkDGn2SChJhSMAct6MPS1IHFzvSEgEEHJBBXMzI9tqAZwE
bGmvUWQ1iZ+cXcQs8fA5yJU+chKRWn/5cfofiAyBweCw3MJDD77jFeVIXAzcHP0U
XJMAchk62nTPKWBQLfshjs+ymL82S6/OGBJpgQR+7uaND1CGiLRKXmflFQfJX7Hq
HH4AGCH1DI7uzzbqVlEdgsMCcWWn1/+dnhIm0UstK01MGmsfDqWpFsdLUCNb50Lh
aEMfM70Z0pUwzuhoYshu7fNKcdeD7OZvTb9hRrIy6daWdWpsJvNRQ58j/GUDCOoj
7Ymws//cOzP3p72+54Ie9lDykuOyr4goeah+u0+nTdKs0Fc+onjH+fwAqNWUlATr
CerJM4oyvQphebePmlx2zdy2Tx44bGbLrfAph7SpEU/KgkaRGt3X1KRPSD9V4sdy
zne0MHuKCqD+AL22Ky5drfHXbhkILbY1g2LEJUs/408B+EdvTXhOw83/QXzsGEb1
/7dlsx0mG7zbIFPBp+uhXuJT2T3DsUhTskmiNnkl
-----END CERTIFICATE-----
`

const argoKey = `-----BEGIN PRIVATE KEY-----
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
FirdKPTDIRLveitPv3lRtneQk49IgYrd
-----END PRIVATE KEY-----
`

func TestDefaultSecureMode(t *testing.T) {

	cmd := NewServerCommand()
	cmd.SetArgs([]string{"--dry-run"})

	// No certs and no explicit secure mode. We should default to insecure
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, b.String(), "secure=false")

	// No certs and explicit insecure mode. We should run insecure
	b = new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--dry-run", "--secure=false"})
	err = cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, b.String(), "secure=false")

	// No certs and explicit secure mode. We should error
	b = new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--dry-run", "--secure"})
	err = cmd.Execute()
	assert.EqualError(t, err, "open argo-server.crt: no such file or directory")

	// Clean up and delete tests files
	defer func() {
		_ = os.Remove("argo-server.crt")
		_ = os.Remove("argo-server.key")
	}()

	_ = ioutil.WriteFile("argo-server.crt", []byte(argoCert), 0o644)
	_ = ioutil.WriteFile("argo-server.key", []byte(argoKey), 0o644)

	// Certs and no explicit secure mode. We should default to secure
	b = new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--dry-run"})
	err = cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, b.String(), "secure=true")

	// Certs and no explicit insecure mode. We should default to insecure
	b = new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--dry-run", "--secure=false"})
	err = cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, b.String(), "secure=false")

	// Certs and explicit secure mode. We should default to secure
	b = new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--dry-run", "--secure=true"})
	err = cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, b.String(), "secure=true")

}
