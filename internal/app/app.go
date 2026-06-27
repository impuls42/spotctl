package app

import (
	"context"
	"fmt"

	"github.com/rackspace-spot/spotctl/internal"
	config "github.com/rackspace-spot/spotctl/pkg"
)

// Context is the shared runtime context for feature operations.
// It centralizes config loading, default org/region resolution, and client creation.
type Context struct {
	Config *config.SpotConfig
	Client *internal.Client

	// Resolved defaults; features can use these if the caller didn't specify.
	Org    string
	Region string
}

type LoadOptions struct {
	Org    string
	Region string

	RequireOrg    bool
	RequireRegion bool
}

func Load(ctx context.Context, opts LoadOptions) (*Context, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load spotctl config; run 'spotctl configure' first: %w", err)
	}

	org := opts.Org
	if org == "" {
		org = cfg.Org
	}
	region := opts.Region
	if region == "" {
		region = cfg.Region
	}

	if opts.RequireOrg && org == "" {
		return nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or provide org")
	}
	if opts.RequireRegion && region == "" {
		return nil, fmt.Errorf("region not specified and not configured; run 'spotctl configure' or provide region")
	}

	client, err := internal.NewClientWithTokens(cfg.RefreshToken, cfg.AccessToken)
	if err != nil {
		return nil, err
	}

	return &Context{
		Config: cfg,
		Client: client,
		Org:    org,
		Region: region,
	}, nil
}
