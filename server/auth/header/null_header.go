package header

import (
	"fmt"

	"google.golang.org/grpc/metadata"

	"github.com/argoproj/argo-workflows/v4/server/auth/types"
)

var NullHeaderAuth Interface = nullService{}

type nullService struct{}

func (n nullService) IsRBACEnabled() bool {
	return false
}

func (n nullService) Authorize(metadata.MD) (*types.Claims, error) {
	return nil, fmt.Errorf("not implemented")
}
