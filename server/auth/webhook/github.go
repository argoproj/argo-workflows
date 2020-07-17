package webhook

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
)

func github(secret string, r *http.Request) error {
	// https://github.com/go-playground/webhooks/blob/v5/github/github.go#L156
	if len(r.Header["X-Github-Event"]) != 1 && len(r.Header["X-Hub-Signature"]) != 1 {
		return NotMatched
	}
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write(peekBody(r))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(r.Header["X-Hub-Signature"][0][5:]), []byte(expectedMAC)) {
		return VerificationFailed
	}
	return nil
}
