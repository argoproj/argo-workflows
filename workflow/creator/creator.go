package creator

import (
	"context"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/v3/server/auth"
	"github.com/argoproj/argo/v3/util/labels"
	"github.com/argoproj/argo/v3/workflow/common"
)

func Label(ctx context.Context, obj metav1.Object) {
	claims := auth.GetClaims(ctx)
	if claims != nil {
		labels.Label(obj, common.LabelKeyCreator, dnsFriendly(claims.Subject))
		if claims.Email != "" {
			labels.Label(obj, common.LabelKeyCreatorEmail, dnsFriendly(strings.Replace(claims.Email, "@", ".at.", 1)))
		}
	}
}

func dnsFriendly(s string) string {
	value := regexp.MustCompile("[^-_.a-z0-9A-Z]").ReplaceAllString(s, "-")
	value = regexp.MustCompile("^[^a-z0-9A-Z]*").ReplaceAllString(value, "")
	value = regexp.MustCompile("[^a-z0-9A-Z]*$").ReplaceAllString(value, "")
	if len(value) > 63 {
		value = value[len(value)-63:]
	}
	return value
}
