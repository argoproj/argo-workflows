package kubeconfig

import (
	"fmt"
	"os"
	"regexp"
)

type tokenVersion = string

const (
	tokenVersion0 tokenVersion = "v0"
	tokenVersion1 tokenVersion = "v1"
	tokenVersion2 tokenVersion = "v2"
)

func getDefaultTokenVersion() tokenVersion {
	value, ok := os.LookupEnv("ARGO_TOKEN_VERSION")
	if !ok {
		return tokenVersion0
	}
	return value
}

func getV2Token() (string, error) {
	token := os.Getenv("ARGO_V2_TOKEN")
	if token == "" {
		return "", fmt.Errorf("no v2 token defined")
	}
	return formatToken(2, token), nil
}

func parseToken(token string) (tokenVersion, string, error) {
	rx := regexp.MustCompile("(v[0-9]):(.*)")
	find := rx.FindStringSubmatch(token)
	if len(find) == 0 {
		return tokenVersion0, "", fmt.Errorf("token not found")
	}
	return find[1], find[2], nil
}

func formatToken(version int, token string) string {
	return fmt.Sprintf("v%d:%s", version, token)
}
