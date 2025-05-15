package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "sigs.k8s.io/yaml/goyaml.v2"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
)

func NewSsoCommand() *cobra.Command {
	var ssoPort int

	cmd := &cobra.Command{
		Use:   "sso",
		Short: "Authenticate with SSO",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSsoFlow(ssoPort)
		},
	}

	cmd.Flags().IntVar(&ssoPort, "sso-port", 8085, "Port to listen for the callback")
	return cmd
}

func runSsoFlow(port int) error {
	if client.ArgoServerOpts.URL == "" {
		return fmt.Errorf("argo server URL is required")
	}

	argoURL, err := url.Parse(client.ArgoServerOpts.URL)
	if err != nil {
		return fmt.Errorf("invalid argo server URL: %w", err)
	}

	baseURL := fmt.Sprintf("%s://%s%s", argoURL.Scheme, argoURL.Host, strings.TrimRight(argoURL.Path, "/"))
	callbackURL := fmt.Sprintf("http://localhost:%d/oauth/callback", port)
	finalRedirectURL := fmt.Sprintf("%s/oauth2/redirect?redirect=%s&cli=true", baseURL, url.QueryEscape(callbackURL))
	exchangeURL := fmt.Sprintf("%s/oauth2/cli/exchange", baseURL)

	fmt.Printf("Opening browser for SSO login: %s\n", finalRedirectURL)

	completion := make(chan string)

	// HTTP server setup
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", makeCallbackHandler(exchangeURL, completion))

	srv := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: mux,
	}

	go func() {
		fmt.Printf("Listening on %s for OAuth2 callback...\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Open browser for login
	if err := open.Start(finalRedirectURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	// Wait for callback
	errMsg := <-completion
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)

	if errMsg != "" {
		return fmt.Errorf("authentication failed: %s", errMsg)
	}

	fmt.Println("✅ Authentication successful")
	return nil
}

func makeCallbackHandler(exchangeURL string, done chan<- string) http.HandlerFunc {
	handleErr := func(w http.ResponseWriter, errMsg string) {
		http.Error(w, html.EscapeString(errMsg), http.StatusBadRequest)
		done <- errMsg
	}

	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("🔐 Received SSO callback")

		code := r.URL.Query().Get("code")
		if code == "" {
			handleErr(w, "no code received in callback")
			return
		}

		fmt.Printf("Code: %s\n", code)

		tokenStr, err := exchangeCode(exchangeURL, code)
		if err != nil {
			handleErr(w, fmt.Sprintf("failed to exchange code: %v", err))
			return
		}

		viper.Set("token", strings.TrimSpace(tokenStr))

		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			// store token in ~/.argo/config
			homeDir, err := os.UserHomeDir()
			if err != nil {
				handleErr(w, fmt.Sprintf("failed to get home directory: %v", err))
				return
			}
			configFile = filepath.Join(homeDir, ".argo", "config.yaml")
		}
		// load yaml from config file
		config := make(map[string]interface{})
		if _, err := os.Stat(configFile); err == nil {
			data, err := os.ReadFile(configFile)
			if err != nil {
				handleErr(w, fmt.Sprintf("failed to read config file: %v", err))
				return
			}
			if err := yaml.Unmarshal(data, &config); err != nil {
				handleErr(w, fmt.Sprintf("failed to unmarshal config file: %v", err))
				return
			}
		}
		// set token in config
		config["token"] = viper.Get("token")
		// marshal config to yaml
		data, err := yaml.Marshal(config)
		if err != nil {
			handleErr(w, fmt.Sprintf("failed to marshal config file: %v", err))
			return
		}
		// write config to file
		if err := os.MkdirAll(filepath.Dir(configFile), os.ModePerm); err != nil {
			handleErr(w, fmt.Sprintf("failed to create config directory: %v", err))
			return
		}
		if err := os.WriteFile(configFile, data, os.ModePerm); err != nil {
			handleErr(w, fmt.Sprintf("failed to write config file: %v", err))
			return
		}

		successPage := `
		<div style="height:100px; width:100%!; display:flex; flex-direction: column; justify-content: center; align-items:center; background-color:#2ecc71; color:white; font-size:22"><div>Authentication successful!</div></div>
		<p style="margin-top:20px; font-size:18; text-align:center">Authentication was successful, you can now return to CLI.</p>
		`
		fmt.Fprint(w, successPage)

		fmt.Println("✅ Code exchanged and saved successfully")
		done <- ""
	}
}

func exchangeCode(exUrl, code string) (string, error) {
	reqURL := fmt.Sprintf("%s?code=%s", exUrl, url.QueryEscape(code))

	// WARNING: This skips certificate validation. Use only in trusted environments!
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
