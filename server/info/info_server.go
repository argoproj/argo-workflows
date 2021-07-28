package info

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v3"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/util/typeassert"
)

type infoServer struct {
	managedNamespace string
	links            []*wfv1.Link
}

func (i *infoServer) GetUserInfo(ctx context.Context, _ *infopkg.GetUserInfoRequest) (*infopkg.GetUserInfoResponse, error) {

	var groupsKey string = "groups"
	claims := auth.GetClaims(ctx)

	// this will be nil for non sso authN and we need to validate
	// this check before we move on with sso claims since
	// nil check with typeassert may cause issues otherwise
	if claims == nil {
		return &infopkg.GetUserInfoResponse{}, nil
	}

	subject, err := typeassert.String(claims["sub"])
	if err != nil {
		return nil, err
	}

	issuer, err := typeassert.String(claims["iss"])
	if err != nil {
		return nil, err
	}

	// check if groups information is supplied via
	//  another custom claim
	if claims["groups"] == nil {
		groupsKey = "groupname"
	}

	fmt.Println(groupsKey)
	groups, err := typeassert.StringSlice(claims[groupsKey])
	if err != nil {
		return nil, err
	}
	fmt.Println(groups)

	email, err := typeassert.String(claims["email"])
	if err != nil {
		return nil, err
	}

	emailVerified, err := typeassert.Bool(claims["email_verified"])
	if err != nil {
		return nil, err
	}

	serviceAccountName, err := typeassert.String(claims["serviceaccount_name"])
	if err != nil {
		return nil, err
	}

	return &infopkg.GetUserInfoResponse{
		Subject:            subject,
		Issuer:             issuer,
		Groups:             groups,
		Email:              email,
		EmailVerified:      emailVerified,
		ServiceAccountName: serviceAccountName,
	}, nil

}

func (i *infoServer) GetInfo(context.Context, *infopkg.GetInfoRequest) (*infopkg.InfoResponse, error) {
	return &infopkg.InfoResponse{ManagedNamespace: i.managedNamespace, Links: i.links}, nil
}

func (i *infoServer) GetVersion(context.Context, *infopkg.GetVersionRequest) (*wfv1.Version, error) {
	version := argo.GetVersion()
	return &version, nil
}

func NewInfoServer(managedNamespace string, links []*wfv1.Link) infopkg.InfoServiceServer {
	return &infoServer{managedNamespace, links}
}
