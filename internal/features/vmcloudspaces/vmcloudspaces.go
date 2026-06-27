package vmcloudspaces

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	rxtspot "github.com/rackspace-spot/spot-go-sdk/api/v1"
	"github.com/rackspace-spot/spotctl/internal/app"
	"k8s.io/klog/v2"
)

func List(ctx context.Context, appCtx *app.Context, org string) (any, error) {
	return appCtx.Client.GetAPI().ListVMCloudSpaces(ctx, org)
}

func Get(ctx context.Context, appCtx *app.Context, org, name string) (*rxtspot.VMCloudSpace, error) {
	return appCtx.Client.GetAPI().GetVMCloudSpace(ctx, org, name)
}

func Delete(ctx context.Context, appCtx *app.Context, org, name string) error {
	return appCtx.Client.GetAPI().DeleteVMCloudSpace(ctx, org, name)
}

func Update(ctx context.Context, appCtx *app.Context, org, name, webhook string) error {
	vmcs := rxtspot.VMCloudSpace{
		Name:    name,
		Webhook: webhook,
	}
	return appCtx.Client.GetAPI().UpdateVMCloudSpace(ctx, org, vmcs)
}

// CreateParams carries a resolved VM cloudspace creation request. Org/Region are
// expected to be resolved by the caller; VMPools already validated. UserData and
// UserDataFromScript are the flag-level fallback applied to pools without their own.
type CreateParams struct {
	Org                string
	Name               string
	Region             string
	Webhook            string
	VMSshKeyRef        rxtspot.VMSshKeyRef
	VMPools            []rxtspot.VMPool
	UserData           string
	UserDataFromScript string
}

// Create creates the VM cloudspace and any inline VM pools. If pool creation is
// cancelled or fails, it deletes the just-created cloudspace before returning. On
// success it returns the freshly fetched cloudspace.
func Create(ctx context.Context, appCtx *app.Context, p CreateParams) (*rxtspot.VMCloudSpace, error) {
	api := appCtx.Client.GetAPI()

	if p.UserData != "" && p.UserDataFromScript != "" {
		return nil, fmt.Errorf("cannot specify both --vm-userdata and --vm-userdata-from-script")
	}
	var finalUserData string
	var err error
	if p.UserDataFromScript != "" {
		finalUserData, err = rxtspot.PrepareUserDataFromScript(p.UserDataFromScript)
		if err != nil {
			return nil, fmt.Errorf("failed to read user data script: %w", err)
		}
	} else if p.UserData != "" {
		finalUserData = rxtspot.PrepareUserData(p.UserData)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("operation cancelled")
	default:
	}

	vmcs := rxtspot.VMCloudSpace{
		Name:        p.Name,
		Org:         p.Org,
		Region:      p.Region,
		Webhook:     p.Webhook,
		VMSshKeyRef: p.VMSshKeyRef,
	}
	if err := api.CreateVMCloudSpace(ctx, vmcs); err != nil {
		return nil, fmt.Errorf("failed to create VM cloudspace: %w", err)
	}

	for _, pool := range p.VMPools {
		select {
		case <-ctx.Done():
			if derr := api.DeleteVMCloudSpace(ctx, p.Org, p.Name); derr != nil {
				klog.Warningf("Failed to clean up VM cloudspace after cancellation: %v", derr)
			}
			return nil, fmt.Errorf("operation cancelled during VM pool creation")
		default:
		}

		if pool.Name == "" {
			pool.Name = uuid.NewString()
		}

		// Use pool-level userdata if set, otherwise fall back to the flag-level userdata.
		poolUserData := pool.VMUserData
		if poolUserData == "" {
			poolUserData = finalUserData
		}

		vmPool := rxtspot.VMPool{
			Name:         pool.Name,
			VMCloudSpace: p.Name,
			ServerClass:  pool.ServerClass,
			BidPrice:     pool.BidPrice,
			Desired:      pool.Desired,
			PoolType:     pool.PoolType,
			VMImage:      pool.VMImage,
			VMUserData:   poolUserData,
		}

		if err := api.CreateVMPool(ctx, p.Org, vmPool); err != nil {
			if delErr := api.DeleteVMCloudSpace(ctx, p.Org, p.Name); delErr != nil {
				klog.Warningf("Failed to clean up VM cloudspace: %v", delErr)
			}
			return nil, fmt.Errorf("failed to create VM pool %s: %w", vmPool.Name, err)
		}
	}

	vmcsResult, err := api.GetVMCloudSpace(ctx, p.Org, p.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM cloudspace after creation: %w", err)
	}
	return vmcsResult, nil
}
