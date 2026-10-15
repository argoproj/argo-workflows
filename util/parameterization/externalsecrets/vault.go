// Package externalsecrets materializes registered external versions as native
// immutable Secrets. It never schedules Workflows or returns values to forms.
package externalsecrets

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// VaultConfig is administrator-owned configuration, never a parameter value.
// A token file can be renewed by Vault Agent without changing this process.
type VaultConfig struct {
	Address              string `json:"address"`
	Mount                string `json:"mount"`
	Entry                string `json:"entry"`
	Field                string `json:"field"`
	TokenFile            string `json:"tokenFile"`
	CAFile               string `json:"caFile,omitempty"`
	AllowHTTPDevelopment bool   `json:"allowHTTPDevelopment,omitempty"`
}
type Version struct {
	Destroyed    bool   `json:"destroyed"`
	DeletionTime string `json:"deletion_time"`
}

func (v Version) Active(now time.Time) bool {
	if v.Destroyed {
		return false
	}
	if v.DeletionTime == "" {
		return true
	}
	at, err := time.Parse(time.RFC3339Nano, v.DeletionTime)
	return err == nil && now.Before(at)
}

type Metadata struct {
	Current  int                `json:"current_version"`
	Versions map[string]Version `json:"versions"`
}
type Provider interface {
	Metadata(context.Context) (Metadata, error)
	Read(context.Context, int) ([]byte, error)
}
type Vault struct {
	config VaultConfig
	client *http.Client
}

func NewVault(c VaultConfig) (*Vault, error) {
	u, err := url.Parse(c.Address)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && (u.Scheme != "http" || !c.AllowHTTPDevelopment)) {
		return nil, fmt.Errorf("external secret backend requires a registered HTTPS origin")
	}
	for _, value := range []string{c.Mount, c.Entry} {
		if value == "" {
			return nil, fmt.Errorf("external secret backend needs mount and entry")
		}
		for segment := range strings.SplitSeq(value, "/") {
			if segment == "" || segment == "." || segment == ".." || strings.ContainsAny(segment, "?%#\\") {
				return nil, fmt.Errorf("invalid external secret path")
			}
		}
	}
	if c.TokenFile == "" || c.Field == "" {
		return nil, fmt.Errorf("external secret backend needs token file and field")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	if c.CAFile != "" {
		pem, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("backend CA is unavailable")
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("backend CA is invalid")
		}
		transport.TLSClientConfig.RootCAs = pool
	}
	return &Vault{c, &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (v *Vault) get(ctx context.Context, kind string, version int, out any) error {
	token, err := os.ReadFile(v.config.TokenFile)
	if err != nil || len(token) > 16384 || len(strings.TrimSpace(string(token))) == 0 {
		return fmt.Errorf("external secret authentication unavailable")
	}
	endpoint := strings.TrimSuffix(v.config.Address, "/") + "/v1/" + v.config.Mount + "/" + kind + "/" + v.config.Entry
	if version > 0 {
		endpoint += "?version=" + strconv.Itoa(version)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("external secret request invalid")
	}
	req.Header.Set("X-Vault-Token", strings.TrimSpace(string(token)))
	response, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("external secret backend unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("external secret read rejected (HTTP %d)", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 131073))
	if err != nil || len(raw) > 131072 || json.Unmarshal(raw, out) != nil {
		return fmt.Errorf("external secret response invalid")
	}
	return nil
}
func (v *Vault) Metadata(ctx context.Context) (Metadata, error) {
	var result struct {
		Data Metadata `json:"data"`
	}
	err := v.get(ctx, "metadata", 0, &result)
	if err == nil && (result.Data.Current < 1 || len(result.Data.Versions) > 1000) {
		err = fmt.Errorf("external secret metadata invalid")
	}
	return result.Data, err
}
func (v *Vault) Read(ctx context.Context, version int) ([]byte, error) {
	var result struct {
		Data struct {
			Data     map[string]json.RawMessage `json:"data"`
			Metadata struct {
				Version      int    `json:"version"`
				Destroyed    bool   `json:"destroyed"`
				DeletionTime string `json:"deletion_time"`
			} `json:"metadata"`
		} `json:"data"`
	}
	if version < 1 {
		return nil, fmt.Errorf("fixed external secret version required")
	}
	if err := v.get(ctx, "data", version, &result); err != nil {
		return nil, err
	}
	meta := result.Data.Metadata
	if meta.Version != version || !(Version{Destroyed: meta.Destroyed, DeletionTime: meta.DeletionTime}).Active(time.Now()) {
		return nil, fmt.Errorf("external secret version revoked or changed")
	}
	var value string
	if json.Unmarshal(result.Data.Data[v.config.Field], &value) != nil || len(value) > 65536 {
		return nil, fmt.Errorf("external secret field must be a string of at most 64 KiB")
	}
	return []byte(value), nil
}
