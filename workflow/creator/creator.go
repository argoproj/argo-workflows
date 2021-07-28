package creator

import (
	"context"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/util/labels"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func Label(ctx context.Context, obj metav1.Object) {
	claims := auth.GetClaims(ctx)
	if len(claims) != 0 {
		labels.Label(obj, common.LabelKeyCreator, dnsFriendly(claims["sub"].(string)))
		// check if emai claim is available
		emailClaim, ok := claims["email"]
		if ok {
			if emailClaim.(string) != "" {
				labels.Label(obj, common.LabelKeyCreatorEmail, dnsFriendly(strings.Replace(emailClaim.(string), "@", ".at.", 1)))
			}
		}
	}
}

// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
func dnsFriendly(s string) string {
	value := regexp.MustCompile("[^-_.a-z0-9A-Z]").ReplaceAllString(s, "-")
	if len(value) > 63 {
		value = value[len(value)-63:]
	}
	value = regexp.MustCompile("^[^a-z0-9A-Z]*").ReplaceAllString(value, "")
	value = regexp.MustCompile("[^a-z0-9A-Z]*$").ReplaceAllString(value, "")
	return value
}
