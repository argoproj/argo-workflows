package webhook

import (
	"net/http"

	"gopkg.in/go-playground/webhooks.v5/github"
)

func github1(secret string, r *http.Request) error {
	// https://github.com/go-playground/webhooks/blob/v5/github/github.go#L156
	if len(r.Header["X-Github-Event"]) != 1 && len(r.Header["X-Hub-Signature"]) != 1 {
		return NotMatched
	}

	hook, _ := github.New(github.Options.Secret(secret))

	_, err := hook.Parse(r, github.PushEvent)

	switch err {
	case github.ErrHMACVerificationFailed:
		return VerificationFailed
	default:
		return err

	}
}
