package organizations

import (
	"context"
	"fmt"

	"github.com/rackspace-spot/spotctl/internal/app"
)

func List(ctx context.Context, appCtx *app.Context) (any, error) {
	return appCtx.Client.GetAPI().ListOrganizations(ctx)
}

// GetByName returns the organization with matching Name, mirroring existing CLI behavior.
func GetByName(ctx context.Context, appCtx *app.Context, name string) (any, error) {
	orgs, err := appCtx.Client.GetAPI().ListOrganizations(ctx)
	if err != nil {
		return nil, err
	}
	for _, org := range orgs {
		if org.Name == name {
			return org, nil
		}
	}
	return nil, fmt.Errorf("organization %q not found", name)
}

