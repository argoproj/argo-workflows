package auth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	infopkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/info"
	"github.com/argoproj/argo-workflows/v4/server/auth/sso"
	"github.com/argoproj/argo-workflows/v4/server/auth/sso/clilogin"
)

const loginSuccessPage = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Argo Workflows</title></head>
<body style="font-family: sans-serif; text-align: center; margin-top: 4em">
<h1>You're logged in</h1><p>You can close this window and return to the terminal.</p>
</body></html>`

// openBrowser is a variable so that tests can replace it.
var openBrowser = func(ctx context.Context, u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", u)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", u)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func NewLoginCommand() *cobra.Command {
	var (
		paste     bool
		noBrowser bool
		timeout   time.Duration
	)
	command := &cobra.Command{
		Use:   "login",
		Short: "Log in to an Argo Server that uses SSO",
		Long: `Log in to an Argo Server that uses SSO, and save the token for later commands.

By default, the CLI opens your browser at the Argo Server's SSO login page and listens on a loopback port for the result.
The Argo Server encrypts the token to a key that only this command holds.
The login must finish within 3 minutes, because the Argo Server's login state expires then.

With --paste, the browser login ends on the User Info page of the UI instead, and you paste the token from that page.
Use --paste when your browser runs on a different machine, for example over SSH, and add --no-browser to only print the login URL.

The CLI saves the token for the server given by --argo-server, --argo-base-href and --secure.
Later commands for that server use the saved token when ARGO_TOKEN isn't set.
"argo auth token" prints it, so other tools can use it too.`,
		Example: `# Log in with your browser
  argo auth login --argo-server argo.example.com:443

# Log in from a machine without a browser
  argo auth login --argo-server argo.example.com:443 --paste --no-browser

# Use the saved token with another tool
  curl -H "Authorization: $(argo auth token --argo-server argo.example.com:443)" https://argo.example.com/api/v1/userinfo`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if client.ArgoServerOpts.URL == "" {
				return errors.New("argo auth login needs the Argo Server address: use --argo-server or set ARGO_SERVER")
			}
			if timeout <= 0 || timeout > clilogin.LoginStateLifetime {
				return fmt.Errorf("--timeout must be more than 0 and at most %s, how long the Argo Server keeps the login state", clilogin.LoginStateLifetime)
			}
			ctx := cmd.Context()
			server := client.ServerKey()
			var payload *clilogin.Payload
			var err error
			if paste {
				payload, err = pasteLogin(ctx, server, !noBrowser, cmd.InOrStdin(), cmd.ErrOrStderr())
			} else {
				payload, err = browserLogin(ctx, server, !noBrowser, timeout, cmd.ErrOrStderr())
			}
			if err != nil {
				return err
			}
			user, err := checkToken(ctx, payload.Token)
			if err != nil {
				return err
			}
			if err = client.SaveToken(server, client.StoredToken{Token: payload.Token, Expiry: payload.Expiry}); err != nil {
				return fmt.Errorf("failed to save the token: %w", err)
			}
			msg := fmt.Sprintf("Logged in to %s as %s", server, user)
			if !payload.Expiry.IsZero() {
				msg += fmt.Sprintf(" until %s", payload.Expiry.Local().Format(time.RFC1123))
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), msg)
			return err
		},
	}
	command.Flags().BoolVar(&paste, "paste", false, "Paste the token from the UI instead of receiving it on a loopback port")
	command.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the login URL without opening a browser")
	command.Flags().DurationVar(&timeout, "timeout", clilogin.LoginStateLifetime, "How long to wait for the browser login to finish, at most 3m")
	return command
}

// tokenCheckTimeout bounds the call that checks a token before it is saved.
const tokenCheckTimeout = 30 * time.Second

// checkToken asks the Argo Server who the token belongs to, so that a bad token
// is never saved. It only accepts SSO tokens: on a server that also allows the
// `server` auth mode, any other value would pass as the server's identity.
func checkToken(ctx context.Context, token string) (string, error) {
	if !strings.HasPrefix(token, sso.Prefix) {
		return "", errors.New("that isn't an Argo Server SSO token, which starts with \"" + sso.Prefix + "\"")
	}
	ctx, cancel := context.WithTimeout(ctx, tokenCheckTimeout)
	defer cancel()
	ctx, apiClient, err := client.NewAPIClientWithAuth(ctx, token)
	if err != nil {
		return "", err
	}
	infoClient, err := apiClient.NewInfoServiceClient()
	if err != nil {
		return "", err
	}
	info, err := infoClient.GetUserInfo(ctx, &infopkg.GetUserInfoRequest{})
	if err != nil {
		return "", fmt.Errorf("the Argo Server rejected the token: %w", err)
	}
	return ssoUser(info)
}

// ssoUser names the user in a GetUserInfo response, and fails unless the
// Argo Server authenticated the request with an SSO token. A server without SSO
// answers any token in `server` mode, with an empty response.
func ssoUser(info *infopkg.GetUserInfoResponse) (string, error) {
	if info.Issuer != sso.Issuer {
		return "", errors.New("the Argo Server didn't accept the token as an SSO login: check that it runs with --auth-mode=sso")
	}
	for _, name := range []string{info.Email, info.Name, info.Subject} {
		if name != "" {
			return name, nil
		}
	}
	return "unknown user", nil
}

func showLoginURL(ctx context.Context, loginURL string, launch bool, stderr io.Writer) {
	if launch {
		if err := openBrowser(ctx, loginURL); err == nil {
			_, _ = fmt.Fprintf(stderr, "Opened your browser. If it didn't open, visit this URL:\n\n  %s\n\n", loginURL)
			return
		}
	}
	_, _ = fmt.Fprintf(stderr, "Visit this URL to log in:\n\n  %s\n\n", loginURL)
}

// browserLogin runs the loopback flow described in package clilogin.
func browserLogin(ctx context.Context, server string, launch bool, timeout time.Duration, stderr io.Writer) (*clilogin.Payload, error) {
	// The callback handler and this goroutine both write to stderr.
	stderr = &lockedWriter{w: stderr}
	session, err := clilogin.NewSession()
	if err != nil {
		return nil, err
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to listen on a loopback port (try --paste): %w", err)
	}
	params, err := session.Params(fmt.Sprintf("http://%s/callback", listener.Addr()))
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	results := make(chan *clilogin.Payload, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		payload, err := session.Open(r.URL.Query())
		if err != nil {
			// Keep waiting: this may be a stray request rather than our login.
			_, _ = fmt.Fprintf(stderr, "Ignored a login callback: %v\n", err)
			http.Error(w, "Login failed: "+err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, loginSuccessPage)
		select {
		case results <- payload:
		default:
		}
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(listener) }()
	defer func() {
		// Shutdown lets the success page finish sending.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	showLoginURL(ctx, server+"/oauth2/redirect?"+params.Encode(), launch, stderr)

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case payload := <-results:
		return payload, nil
	case <-timer.C:
		return nil, fmt.Errorf("timed out after %s waiting for the browser login. If your browser runs on another machine, use --paste", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// pasteLogin sends you to the UI's User Info page, then reads the token you paste.
func pasteLogin(ctx context.Context, server string, launch bool, stdin io.Reader, stderr io.Writer) (*clilogin.Payload, error) {
	userInfoPath := path.Join("/", client.ArgoServerOpts.Path, "userinfo")
	showLoginURL(ctx, server+"/oauth2/redirect?"+url.Values{"redirect": {userInfoPath}}.Encode(), launch, stderr)
	_, _ = fmt.Fprint(stderr, "After you log in, copy the ARGO_TOKEN value from the User Info page and paste it here: ")
	line, err := readSecretLine(stdin, stderr)
	if err != nil {
		return nil, err
	}
	token, err := normalizeToken(line)
	if err != nil {
		return nil, err
	}
	return &clilogin.Payload{Token: token}, nil
}

// normalizeToken accepts the token as the UI shows it, with or without the
// `export ARGO_TOKEN=` prefix, quotes and the `Bearer ` prefix.
func normalizeToken(s string) (string, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "export ")
	s = strings.TrimPrefix(s, "ARGO_TOKEN=")
	s = strings.Trim(s, `'"`)
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("no token was pasted")
	}
	if strings.ContainsAny(s, " \t") && !strings.HasPrefix(s, "Bearer ") {
		return "", errors.New("that doesn't look like an Argo token")
	}
	if !strings.HasPrefix(s, "Bearer ") {
		s = "Bearer " + s
	}
	if !strings.HasPrefix(s, sso.Prefix) {
		return "", errors.New("that isn't an Argo Server SSO token: copy the ARGO_TOKEN value from the User Info page, which starts with \"" + sso.Prefix + "\"")
	}
	return s, nil
}

// readSecretLine reads one line without echoing it. On a terminal it reads in
// raw mode, because canonical mode truncates long lines (macOS limits a line to
// 1024 bytes) and SSO tokens can be longer.
func readSecretLine(stdin io.Reader, stderr io.Writer) (string, error) {
	f, ok := stdin.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && (!errors.Is(err, io.EOF) || line == "") {
			return "", fmt.Errorf("failed to read the token: %w", err)
		}
		return line, nil
	}
	fd := int(f.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(fd, state) }()
	var line []byte
	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if err != nil {
			return "", err
		}
		for _, c := range buf[:n] {
			switch c {
			case '\r', '\n':
				_, _ = fmt.Fprint(stderr, "\r\n")
				return string(line), nil
			case 3: // Ctrl-C
				_, _ = fmt.Fprint(stderr, "\r\n")
				return "", errors.New("interrupted")
			case 4: // Ctrl-D
				if len(line) == 0 {
					return "", io.EOF
				}
			case 8, 127: // Backspace
				if len(line) > 0 {
					line = line[:len(line)-1]
				}
			default:
				line = append(line, c)
			}
		}
	}
}

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
