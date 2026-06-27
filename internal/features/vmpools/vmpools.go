package vmpools

import (
	"context"
	"fmt"

	rxtspot "github.com/rackspace-spot/spot-go-sdk/api/v1"
	"github.com/rackspace-spot/spotctl/internal/app"
)

func List(ctx context.Context, appCtx *app.Context, org, vmCloudSpace string) (any, error) {
	return appCtx.Client.GetAPI().ListVMPools(ctx, org, vmCloudSpace)
}

func Get(ctx context.Context, appCtx *app.Context, org, name string) (any, error) {
	return appCtx.Client.GetAPI().GetVMPool(ctx, org, name)
}

func Delete(ctx context.Context, appCtx *app.Context, org, name string) error {
	return appCtx.Client.GetAPI().DeleteVMPool(ctx, org, name)
}

// CreateParams carries already-validated values for pool creation. BidPrice and
// Desired are expected to be validated by the caller; UserData/UserDataFromScript
// are raw inputs handled here (mutually exclusive).
type CreateParams struct {
	Org                string
	Name               string
	VMCloudSpace       string
	ServerClass        string
	BidPrice           string
	Desired            int
	PoolType           string
	VMImage            string
	UserData           string
	UserDataFromScript string
}

// Create builds the VMPool (resolving cloud-init user data), submits it, and
// returns the submitted pool so the caller can present its details.
func Create(ctx context.Context, appCtx *app.Context, p CreateParams) (rxtspot.VMPool, error) {
	if p.UserData != "" && p.UserDataFromScript != "" {
		return rxtspot.VMPool{}, fmt.Errorf("cannot specify both --vm-userdata and --vm-userdata-from-script")
	}

	var finalUserData string
	var err error
	if p.UserDataFromScript != "" {
		finalUserData, err = rxtspot.PrepareUserDataFromScript(p.UserDataFromScript)
		if err != nil {
			return rxtspot.VMPool{}, fmt.Errorf("failed to read user data script: %w", err)
		}
	} else if p.UserData != "" {
		finalUserData = rxtspot.PrepareUserData(p.UserData)
	}

	pool := rxtspot.VMPool{
		Name:         p.Name,
		VMCloudSpace: p.VMCloudSpace,
		ServerClass:  p.ServerClass,
		BidPrice:     p.BidPrice,
		Desired:      p.Desired,
		PoolType:     p.PoolType,
		VMImage:      p.VMImage,
		VMUserData:   finalUserData,
	}

	if err := appCtx.Client.GetAPI().CreateVMPool(ctx, p.Org, pool); err != nil {
		return rxtspot.VMPool{}, err
	}
	return pool, nil
}

// UpdateParams carries an update request. Desired < 0 leaves the count unchanged;
// an empty BidPrice (already validated when set) leaves the price unchanged.
type UpdateParams struct {
	Org      string
	Name     string
	Desired  int
	BidPrice string
}

func Update(ctx context.Context, appCtx *app.Context, p UpdateParams) error {
	pool := rxtspot.VMPool{Name: p.Name}
	if p.Desired >= 0 {
		pool.Desired = p.Desired
	}
	if p.BidPrice != "" {
		pool.BidPrice = p.BidPrice
	}
	return appCtx.Client.GetAPI().UpdateVMPool(ctx, p.Org, pool)
}
