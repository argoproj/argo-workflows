package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StoredToken is an Argo Server token saved by `argo auth login`.
type StoredToken struct {
	Token string `json:"token"`
	// Expiry is zero when the CLI couldn't learn it, e.g. for a pasted token.
	Expiry time.Time `json:"expiry,omitzero"`
}

// Expired reports whether the token is known to have expired at now.
func (t StoredToken) Expired(now time.Time) bool {
	return !t.Expiry.IsZero() && !now.Before(t.Expiry)
}

// tokenStorePath is a variable so that tests can point it at a temporary file.
var tokenStorePath = func() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "argo", "tokens.json"), nil
}

// ServerKey identifies the Argo Server selected by --argo-server,
// --argo-base-href and --secure, for keying saved tokens.
func ServerKey() string {
	return strings.TrimSuffix(ArgoServerOpts.GetURL(), "/")
}

func readTokenStore() (map[string]StoredToken, string, error) {
	path, err := tokenStorePath()
	if err != nil {
		return nil, "", err
	}
	tokens := map[string]StoredToken{}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return tokens, path, nil
	}
	if err != nil {
		return nil, path, err
	}
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, path, fmt.Errorf("failed to read saved tokens from %s: %w", path, err)
	}
	return tokens, path, nil
}

func writeTokenStore(path string, tokens map[string]StoredToken) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tokens-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	// CreateTemp creates the file with mode 0600.
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// LoadToken returns the saved token for server, or nil if there isn't one.
func LoadToken(server string) (*StoredToken, error) {
	tokens, _, err := readTokenStore()
	if err != nil {
		return nil, err
	}
	t, ok := tokens[server]
	if !ok {
		return nil, nil
	}
	return &t, nil
}

// SaveToken saves the token for server, replacing any previous one.
func SaveToken(server string, t StoredToken) error {
	tokens, path, err := readTokenStore()
	if err != nil {
		return err
	}
	tokens[server] = t
	return writeTokenStore(path, tokens)
}

// DeleteToken removes the saved token for server and reports whether there was one.
func DeleteToken(server string) (bool, error) {
	tokens, path, err := readTokenStore()
	if err != nil {
		return false, err
	}
	if _, ok := tokens[server]; !ok {
		return false, nil
	}
	delete(tokens, server)
	return true, writeTokenStore(path, tokens)
}
