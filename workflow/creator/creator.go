package creator

import (
	"context"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/labels"
	"github.com/argoproj/argo/workflow/common"
)

func Label(ctx context.Context, obj metav1.Object) {
	claims := auth.GetClaims(ctx)
	if claims != nil {
		value := regexp.MustCompile("[^-_.a-z0-9A-Z]").ReplaceAllString(claims.Subject, "-")
		if len(value) > 63 {
			value = value[len(value)-63:]
		}
		value = strings.TrimLeft(value, "-")
		labels.Label(obj, common.LabelKeyCreator, value)
	}
}
