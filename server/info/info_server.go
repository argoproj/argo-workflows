package info

import (
	"context"
	"os"

	"github.com/argoproj/argo-workflows/v3"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
)

type infoServer struct {
	managedNamespace string
	links            []*wfv1.Link
	navColor         string
}

func (i *infoServer) GetUserInfo(ctx context.Context, _ *infopkg.GetUserInfoRequest) (*infopkg.GetUserInfoResponse, error) {
	claims := auth.GetClaims(ctx)
	if claims != nil {
		return &infopkg.GetUserInfoResponse{
			Subject:            claims.Subject,
			Issuer:             claims.Issuer,
			Groups:             claims.Groups,
			Email:              claims.Email,
			EmailVerified:      claims.EmailVerified,
			ServiceAccountName: claims.ServiceAccountName,
		}, nil
	}
	return &infopkg.GetUserInfoResponse{}, nil
}

func (i *infoServer) GetInfo(context.Context, *infopkg.GetInfoRequest) (*infopkg.InfoResponse, error) {
	modals := map[string]bool{
		"feedback":      os.Getenv("FEEDBACK_MODAL") != "false",
		"firstTimeUser": os.Getenv("FIRST_TIME_USER_MODAL") != "false",
		"newVersion":    os.Getenv("NEW_VERSION_MODAL") != "false",
	}
	return &infopkg.InfoResponse{
		ManagedNamespace: i.managedNamespace,
		Links:            i.links,
		Modals:           modals,
		NavColor:         i.navColor,
	}, nil
}

func (i *infoServer) GetVersion(context.Context, *infopkg.GetVersionRequest) (*wfv1.Version, error) {
	version := argo.GetVersion()
	return &version, nil
}

func NewInfoServer(managedNamespace string, links []*wfv1.Link, navColor string) infopkg.InfoServiceServer {
	return &infoServer{managedNamespace, links, navColor}
}
