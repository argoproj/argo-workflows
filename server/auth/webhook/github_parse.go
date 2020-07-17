package webhook

import (
	"net/http"

	"gopkg.in/go-playground/webhooks.v5/github"
)

func githubParse(secret string, r *http.Request) error {
	hook, err := github.New(github.Options.Secret(secret))
	if err != nil {
		return err
	}
	_, err = hook.Parse(r, github.PushEvent)
	switch err {
	case github.ErrHMACVerificationFailed:
		return VerificationFailed
	default:
		return err
	}
}
