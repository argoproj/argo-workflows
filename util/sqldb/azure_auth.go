package sqldb

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/lib/pq"
)

type azureConnector struct {
	dsn   string
	scope string
}

func (c *azureConnector) Connect(ctx context.Context) (driver.Conn, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain a credential: %w", err)
	}

	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{c.scope}})
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Escape single quotes in token just in case
	escapedToken := strings.ReplaceAll(token.Token, "'", "\\'")

	// Append password to DSN
	dsnWithPassword := fmt.Sprintf("%s password='%s'", c.dsn, escapedToken)

	return pq.Driver{}.Open(dsnWithPassword)
}

func (c *azureConnector) Driver() driver.Driver {
	return pq.Driver{}
}
