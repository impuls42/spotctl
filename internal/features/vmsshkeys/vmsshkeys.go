package vmsshkeys

import (
	"context"

	rxtspot "github.com/rackspace-spot/spot-go-sdk/api/v1"
	"github.com/rackspace-spot/spotctl/internal/app"
)

func List(ctx context.Context, appCtx *app.Context, org string) (any, error) {
	return appCtx.Client.GetAPI().ListVMSSHKeys(ctx, org)
}

func Get(ctx context.Context, appCtx *app.Context, org, name string) (any, error) {
	return appCtx.Client.GetAPI().GetVMSSHKey(ctx, org, name)
}

type CreateParams struct {
	Org         string
	Name        string
	PublicKey   string
	Description string
}

func Create(ctx context.Context, appCtx *app.Context, params CreateParams) error {
	key := rxtspot.VMSSHKey{
		Name:        params.Name,
		Org:         params.Org,
		PublicKey:   params.PublicKey,
		Description: params.Description,
	}
	return appCtx.Client.GetAPI().CreateVMSSHKey(ctx, key)
}

func Delete(ctx context.Context, appCtx *app.Context, org, name string) error {
	return appCtx.Client.GetAPI().DeleteVMSSHKey(ctx, org, name)
}
