package webhook

import (
	"net/http"

	"gopkg.in/go-playground/webhooks.v5/github"
)

func githubParse(secret string, r *http.Request) bool {
	hook, err := github.New(github.Options.Secret(secret))
	if err != nil {
		return false
	}
	_, err = hook.Parse(r, github.PushEvent)
	return err == nil
}
