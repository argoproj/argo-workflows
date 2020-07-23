package creator

import (
	"context"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/labels"
	"github.com/argoproj/argo/workflow/common"
)

func Label(ctx context.Context, obj metav1.Object) {
	claims := auth.GetClaimSet(ctx)
	if claims != nil {
		labels.Label(obj, common.LabelKeyCreator, regexp.MustCompile("[^-_.a-z0-9A-Z]").ReplaceAllString(claims.Sub, "-"))
	}
}
