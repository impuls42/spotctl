package cloudspaces

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	rxtspot "github.com/rackspace-spot/spot-go-sdk/api/v1"
	"github.com/rackspace-spot/spotctl/internal/app"
)

type SpotNodePoolParams struct {
	Name        string            `json:"name,omitempty" yaml:"name,omitempty"`
	ServerClass string            `json:"serverclass" yaml:"serverclass"`
	Desired     int               `json:"desired" yaml:"desired"`
	BidPrice    string            `json:"bidprice" yaml:"bidprice"`
	Labels      map[string]string `json:"customLabels,omitempty" yaml:"customLabels,omitempty"`
	Annotations map[string]string `json:"customAnnotations,omitempty" yaml:"customAnnotations,omitempty"`
}

type OnDemandNodePoolParams struct {
	Name        string            `json:"name,omitempty" yaml:"name,omitempty"`
	ServerClass string            `json:"serverclass" yaml:"serverclass"`
	Desired     int               `json:"desired" yaml:"desired"`
	Labels      map[string]string `json:"customLabels,omitempty" yaml:"customLabels,omitempty"`
	Annotations map[string]string `json:"customAnnotations,omitempty" yaml:"customAnnotations,omitempty"`
}

type CreateParams struct {
	Name                 string `json:"name" yaml:"name"`
	Org                  string `json:"org,omitempty" yaml:"org,omitempty"`
	Region               string `json:"region,omitempty" yaml:"region,omitempty"`
	KubernetesVersion    string `json:"kubernetesVersion,omitempty" yaml:"kubernetesVersion,omitempty"`
	PreemptionWebhookURL string `json:"preemptionWebhookURL,omitempty" yaml:"preemptionWebhookURL,omitempty"`
	CNI                  string `json:"cni,omitempty" yaml:"cni,omitempty"`

	SpotNodePools     []SpotNodePoolParams     `json:"spotNodePools,omitempty" yaml:"spotNodePools,omitempty"`
	OnDemandNodePools []OnDemandNodePoolParams `json:"onDemandNodePools,omitempty" yaml:"onDemandNodePools,omitempty"`
}

func List(ctx context.Context, appCtx *app.Context, org string) (any, error) {
	return appCtx.Client.GetAPI().ListCloudspaces(ctx, org)
}

func Get(ctx context.Context, appCtx *app.Context, org, name string) (any, error) {
	return appCtx.Client.GetAPI().GetCloudspace(ctx, org, name)
}

func Delete(ctx context.Context, appCtx *app.Context, org, name string) error {
	return appCtx.Client.GetAPI().DeleteCloudspace(ctx, org, name)
}

func GetConfig(ctx context.Context, appCtx *app.Context, org, name string) (string, error) {
	return appCtx.Client.GetAPI().GetCloudspaceConfig(ctx, org, name)
}

func Create(ctx context.Context, appCtx *app.Context, params CreateParams) (any, error) {
	if params.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return nil, fmt.Errorf("organization not specified and not configured")
	}
	region := params.Region
	if region == "" {
		region = appCtx.Region
	}
	if region == "" {
		return nil, fmt.Errorf("region not specified and not configured")
	}

	k8sVersion := params.KubernetesVersion
	if k8sVersion == "" {
		k8sVersion = "1.31.1"
	}
	cni := params.CNI
	if cni == "" {
		cni = "calico"
	}

	// Validate pools before creating the cloudspace.
	for _, pool := range params.SpotNodePools {
		if pool.ServerClass == "" || pool.Desired <= 0 || pool.BidPrice == "" {
			return nil, fmt.Errorf("spot node pool requires serverclass, desired (>0), and bidprice")
		}
	}
	for _, pool := range params.OnDemandNodePools {
		if pool.ServerClass == "" || pool.Desired <= 0 {
			return nil, fmt.Errorf("on-demand node pool requires serverclass and desired (>0)")
		}
	}

	cloudspace := rxtspot.CloudSpace{
		Name:                 params.Name,
		Org:                  org,
		Region:               region,
		KubernetesVersion:    k8sVersion,
		CNI:                  cni,
		PreemptionWebhookURL: params.PreemptionWebhookURL,
	}

	if err := appCtx.Client.GetAPI().CreateCloudspace(ctx, cloudspace); err != nil {
		return nil, err
	}

	cleanupCloudspace := func() {
		_ = appCtx.Client.GetAPI().DeleteCloudspace(context.Background(), org, params.Name)
	}

	// Create spot node pools if any
	for _, pool := range params.SpotNodePools {
		select {
		case <-ctx.Done():
			cleanupCloudspace()
			return nil, ctx.Err()
		default:
		}
		name := pool.Name
		if name == "" {
			name = uuid.NewString()
		}
		p := rxtspot.SpotNodePool{
			Name:              name,
			Org:               org,
			Cloudspace:        params.Name,
			ServerClass:       pool.ServerClass,
			BidPrice:          pool.BidPrice,
			Desired:           pool.Desired,
			CustomLabels:      pool.Labels,
			CustomAnnotations: pool.Annotations,
		}
		if err := appCtx.Client.GetAPI().CreateSpotNodePool(ctx, org, p); err != nil {
			cleanupCloudspace()
			return nil, fmt.Errorf("failed creating spot node pool %s: %w", p.Name, err)
		}
	}

	// Create on-demand node pools if any
	for _, pool := range params.OnDemandNodePools {
		select {
		case <-ctx.Done():
			cleanupCloudspace()
			return nil, ctx.Err()
		default:
		}
		name := pool.Name
		if name == "" {
			name = uuid.NewString()
		}
		p := rxtspot.OnDemandNodePool{
			Name:              name,
			Org:               org,
			Cloudspace:        params.Name,
			ServerClass:       pool.ServerClass,
			Desired:           pool.Desired,
			CustomLabels:      pool.Labels,
			CustomAnnotations: pool.Annotations,
		}
		if err := appCtx.Client.GetAPI().CreateOnDemandNodePool(ctx, org, p); err != nil {
			cleanupCloudspace()
			return nil, fmt.Errorf("failed creating on-demand node pool %s: %w", p.Name, err)
		}
	}

	return appCtx.Client.GetAPI().GetCloudspace(ctx, org, params.Name)
}

