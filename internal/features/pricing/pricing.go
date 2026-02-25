package pricing

import (
	"context"

	"github.com/rackspace-spot/spotctl/internal/app"
)

func GetAll(ctx context.Context, appCtx *app.Context) (any, error) {
	return appCtx.Client.GetAPI().GetPriceDetails(ctx)
}

func GetForServerClass(ctx context.Context, appCtx *app.Context, serverClass string) (any, error) {
	return appCtx.Client.GetAPI().GetPriceDetailsForServerClass(ctx, serverClass)
}

