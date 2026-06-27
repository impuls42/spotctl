package serverclasses

import (
	"context"

	"github.com/rackspace-spot/spotctl/internal/app"
)

func List(ctx context.Context, appCtx *app.Context, region string) (any, error) {
	return appCtx.Client.GetAPI().ListServerClasses(ctx, region)
}

func Get(ctx context.Context, appCtx *app.Context, name string) (any, error) {
	return appCtx.Client.GetAPI().GetServerClass(ctx, name)
}
