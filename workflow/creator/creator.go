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

type ActionType string

const (
	ActionUpdate    ActionType = "Update"
	ActionSuspend   ActionType = "Suspend"
	ActionStop      ActionType = "Stop"
	ActionTerminate ActionType = "Terminate"
	ActionResume    ActionType = "Resume"
	ActionNone      ActionType = ""
)

func Label(ctx context.Context, obj metav1.Object, userLabelKey string, userEmailLabelKey string, preferredUsernameLabelKey string, action ActionType) {
	claims := auth.GetClaims(ctx)
	if claims != nil {
		if claims.Subject != "" {
			labels.Label(obj, userLabelKey, dnsFriendly(claims.Subject))
		} else {
			labels.UnLabel(obj, userLabelKey)
		}
		if claims.Email != "" {
			labels.Label(obj, userEmailLabelKey, dnsFriendly(strings.Replace(claims.Email, "@", ".at.", 1)))
		} else {
			labels.UnLabel(obj, userEmailLabelKey)
		}
		if claims.PreferredUsername != "" {
			labels.Label(obj, preferredUsernameLabelKey, dnsFriendly(claims.PreferredUsername))
		} else {
			labels.UnLabel(obj, preferredUsernameLabelKey)
		}
		if action != "" {
			labels.Label(obj, common.LabelKeyAction, dnsFriendly(string(action)))
		} else {
			labels.UnLabel(obj, common.LabelKeyAction)
		}
	} else {
		// If the object already has creator-related labels, but the actual request lacks auth information,
		// remove the creator-related labels from the object.
		labels.UnLabel(obj, userLabelKey)
		labels.UnLabel(obj, userEmailLabelKey)
		labels.UnLabel(obj, preferredUsernameLabelKey)
		labels.UnLabel(obj, common.LabelKeyAction)
	}
}

func LabelCreator(ctx context.Context, obj metav1.Object) {
	Label(ctx, obj, common.LabelKeyCreator, common.LabelKeyCreatorEmail, common.LabelKeyCreatorPreferredUsername, "")
}

func LabelActor(ctx context.Context, obj metav1.Object, action ActionType) {
	Label(ctx, obj, common.LabelKeyActor, common.LabelKeyActorEmail, common.LabelKeyActorPreferredUsername, action)
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

func UserInfoMap(ctx context.Context) map[string]string {
	claims := auth.GetClaims(ctx)
	if claims == nil {
		return nil
	}
	res := map[string]string{}
	if claims.Subject != "" {
		res["User"] = claims.Subject
	}
	if claims.Email != "" {
		res["Email"] = claims.Email
	}
	if claims.PreferredUsername != "" {
		res["PreferredUsername"] = claims.PreferredUsername
	}
	return res
}

func UserActionLabel(ctx context.Context, action ActionType) map[string]string {
	claims := auth.GetClaims(ctx)
	if claims == nil {
		return nil
	}
	res := map[string]string{}
	if claims.Subject != "" {
		res[common.LabelKeyActor] = dnsFriendly(claims.Subject)
	}
	if claims.Email != "" {
		res[common.LabelKeyActorEmail] = dnsFriendly(claims.Email)
	}
	if claims.PreferredUsername != "" {
		res[common.LabelKeyActorPreferredUsername] = dnsFriendly(claims.PreferredUsername)
	}
	if action != ActionNone {
		res[common.LabelKeyAction] = string(action)
	}
	return res
}
