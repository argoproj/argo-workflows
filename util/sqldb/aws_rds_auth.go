package sqldb

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/lib/pq"
)

type awsRDSConnector struct {
	dsn      string
	endpoint string
	username string
	region   string
}

func (c *awsRDSConnector) Connect(ctx context.Context) (driver.Conn, error) {
	opts := []func(*awsconfig.LoadOptions) error{}
	if c.region != "" {
		opts = append(opts, awsconfig.WithRegion(c.region))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	token, err := auth.BuildAuthToken(ctx, c.endpoint, awsCfg.Region, c.username, awsCfg.Credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to build RDS auth token: %w", err)
	}

	// Escape single quotes in token for safe DSN interpolation
	escapedToken := strings.ReplaceAll(token, "'", "\\'")

	dsnWithPassword := fmt.Sprintf("%s password='%s'", c.dsn, escapedToken)

	return pq.Driver{}.Open(dsnWithPassword)
}

func (c *awsRDSConnector) Driver() driver.Driver {
	return pq.Driver{}
}
