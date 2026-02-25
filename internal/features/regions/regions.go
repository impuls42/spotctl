package regions

import (
	"context"

	rxtspot "github.com/rackspace-spot/spot-go-sdk/api/v1"
	"github.com/rackspace-spot/spotctl/internal/app"
)

func List(ctx context.Context, appCtx *app.Context) ([]rxtspot.Region, error) {
	return appCtx.Client.GetAPI().ListRegions(ctx)
}

func Get(ctx context.Context, appCtx *app.Context, name string) (*rxtspot.Region, error) {
	return appCtx.Client.GetAPI().GetRegion(ctx, name)
}

