package nodepools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	rxtspot "github.com/rackspace-spot/spot-go-sdk/api/v1"
	"github.com/rackspace-spot/spotctl/internal/app"
)

func ParseKVCommaSeparated(s string) (map[string]string, error) {
	out := map[string]string{}
	if strings.TrimSpace(s) == "" {
		return out, nil
	}
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid pair %q (expected key=value)", pair)
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k == "" {
			return nil, fmt.Errorf("empty key in %q", pair)
		}
		out[k] = v
	}
	return out, nil
}

// ---- Spot nodepools ----

type SpotListParams struct {
	Org        string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Cloudspace string `json:"cloudspace" jsonschema:"Cloudspace name"`
}

type SpotGetParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"Spot node pool name (UUID)"`
}

type SpotCreateParams struct {
	Org               string            `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Cloudspace        string            `json:"cloudspace" jsonschema:"Cloudspace name"`
	ServerClass       string            `json:"serverclass" jsonschema:"Server class"`
	Desired           int               `json:"desired" jsonschema:"Desired number of nodes"`
	BidPrice          string            `json:"bidprice" jsonschema:"Maximum bid price"`
	CustomLabels      map[string]string `json:"customLabels,omitempty" jsonschema:"Custom labels for the node pool"`
	CustomAnnotations map[string]string `json:"customAnnotations,omitempty" jsonschema:"Custom annotations for the node pool"`
	Name              string            `json:"name,omitempty" jsonschema:"Optional explicit node pool name (UUID); generated if empty"`
}

type SpotUpdateParams struct {
	Org               string            `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name              string            `json:"name" jsonschema:"Spot node pool name (UUID)"`
	Cloudspace        string            `json:"cloudspace" jsonschema:"Cloudspace name"`
	Desired           *int              `json:"desired,omitempty" jsonschema:"Desired number of nodes; if omitted, unchanged"`
	BidPrice          *string           `json:"bidprice,omitempty" jsonschema:"Maximum bid price; if omitted, unchanged"`
	CustomLabels      map[string]string `json:"customLabels,omitempty" jsonschema:"Custom labels for the node pool; if omitted, unchanged"`
	CustomAnnotations map[string]string `json:"customAnnotations,omitempty" jsonschema:"Custom annotations for the node pool; if omitted, unchanged"`
}

type SpotDeleteParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"Spot node pool name (UUID)"`
}

func SpotList(ctx context.Context, appCtx *app.Context, params SpotListParams) (any, error) {
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return nil, fmt.Errorf("organization not specified and not configured")
	}
	if params.Cloudspace == "" {
		return nil, fmt.Errorf("cloudspace is required")
	}
	return appCtx.Client.GetAPI().ListSpotNodePools(ctx, org, params.Cloudspace)
}

func SpotGet(ctx context.Context, appCtx *app.Context, params SpotGetParams) (any, error) {
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return nil, fmt.Errorf("organization not specified and not configured")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return appCtx.Client.GetAPI().GetSpotNodePool(ctx, org, params.Name)
}

func SpotCreate(ctx context.Context, appCtx *app.Context, params SpotCreateParams) (any, error) {
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return nil, fmt.Errorf("organization not specified and not configured")
	}
	if params.Cloudspace == "" || params.ServerClass == "" || params.Desired <= 0 || params.BidPrice == "" {
		return nil, fmt.Errorf("cloudspace, serverclass, desired (>0), and bidprice are required")
	}

	name := params.Name
	if name == "" {
		name = uuid.NewString()
	}

	pool := rxtspot.SpotNodePool{
		Name:              name,
		Org:               org,
		Cloudspace:        params.Cloudspace,
		ServerClass:       params.ServerClass,
		Desired:           params.Desired,
		BidPrice:          params.BidPrice,
		CustomLabels:      params.CustomLabels,
		CustomAnnotations: params.CustomAnnotations,
	}

	if err := appCtx.Client.GetAPI().CreateSpotNodePool(ctx, org, pool); err != nil {
		return nil, err
	}
	return appCtx.Client.GetAPI().GetSpotNodePool(ctx, org, name)
}

func SpotUpdate(ctx context.Context, appCtx *app.Context, params SpotUpdateParams) (any, error) {
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return nil, fmt.Errorf("organization not specified and not configured")
	}
	if params.Name == "" || params.Cloudspace == "" {
		return nil, fmt.Errorf("name and cloudspace are required")
	}

	current, err := appCtx.Client.GetAPI().GetSpotNodePool(ctx, org, params.Name)
	if err != nil {
		return nil, err
	}

	// Preserve current fields unless explicitly overridden.
	updated := rxtspot.SpotNodePool{
		Name:       current.Name,
		Org:        org,
		Cloudspace: params.Cloudspace,
		// Keep serverclass from current; CLI doesn't allow changing it.
		ServerClass:       current.ServerClass,
		Desired:           current.Desired,
		BidPrice:          current.BidPrice,
		CustomLabels:      current.CustomLabels,
		CustomAnnotations: current.CustomAnnotations,
	}

	if params.Desired != nil {
		updated.Desired = *params.Desired
	}
	if params.BidPrice != nil {
		updated.BidPrice = *params.BidPrice
	}
	if params.CustomLabels != nil {
		updated.CustomLabels = params.CustomLabels
	}
	if params.CustomAnnotations != nil {
		updated.CustomAnnotations = params.CustomAnnotations
	}

	if err := appCtx.Client.GetAPI().UpdateSpotNodePool(ctx, org, updated); err != nil {
		return nil, err
	}
	return appCtx.Client.GetAPI().GetSpotNodePool(ctx, org, params.Name)
}

func SpotDelete(ctx context.Context, appCtx *app.Context, params SpotDeleteParams) error {
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return fmt.Errorf("organization not specified and not configured")
	}
	if params.Name == "" {
		return fmt.Errorf("name is required")
	}
	return appCtx.Client.GetAPI().DeleteSpotNodePool(ctx, org, params.Name)
}

// ---- On-demand nodepools ----

type OnDemandListParams struct {
	Org        string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Cloudspace string `json:"cloudspace" jsonschema:"Cloudspace name"`
}

type OnDemandGetParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"On-demand node pool name (UUID)"`
}

type OnDemandCreateParams struct {
	Org               string            `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Cloudspace        string            `json:"cloudspace" jsonschema:"Cloudspace name"`
	ServerClass       string            `json:"serverclass" jsonschema:"Server class"`
	Desired           int               `json:"desired" jsonschema:"Desired number of nodes"`
	CustomLabels      map[string]string `json:"customLabels,omitempty" jsonschema:"Custom labels for the node pool"`
	CustomAnnotations map[string]string `json:"customAnnotations,omitempty" jsonschema:"Custom annotations for the node pool"`
	Name              string            `json:"name,omitempty" jsonschema:"Optional explicit node pool name (UUID); generated if empty"`
}

type OnDemandUpdateParams struct {
	Org               string            `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name              string            `json:"name" jsonschema:"On-demand node pool name (UUID)"`
	Cloudspace        string            `json:"cloudspace" jsonschema:"Cloudspace name"`
	Desired           *int              `json:"desired,omitempty" jsonschema:"Desired number of nodes; if omitted, unchanged"`
	CustomLabels      map[string]string `json:"customLabels,omitempty" jsonschema:"Custom labels for the node pool; if omitted, unchanged"`
	CustomAnnotations map[string]string `json:"customAnnotations,omitempty" jsonschema:"Custom annotations for the node pool; if omitted, unchanged"`
}

type OnDemandDeleteParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"On-demand node pool name (UUID)"`
}

func OnDemandList(ctx context.Context, appCtx *app.Context, params OnDemandListParams) (any, error) {
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return nil, fmt.Errorf("organization not specified and not configured")
	}
	if params.Cloudspace == "" {
		return nil, fmt.Errorf("cloudspace is required")
	}
	return appCtx.Client.GetAPI().ListOnDemandNodePools(ctx, org, params.Cloudspace)
}

func OnDemandGet(ctx context.Context, appCtx *app.Context, params OnDemandGetParams) (any, error) {
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return nil, fmt.Errorf("organization not specified and not configured")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return appCtx.Client.GetAPI().GetOnDemandNodePool(ctx, org, params.Name)
}

func OnDemandCreate(ctx context.Context, appCtx *app.Context, params OnDemandCreateParams) (any, error) {
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return nil, fmt.Errorf("organization not specified and not configured")
	}
	if params.Cloudspace == "" || params.ServerClass == "" || params.Desired <= 0 {
		return nil, fmt.Errorf("cloudspace, serverclass, and desired (>0) are required")
	}

	name := params.Name
	if name == "" {
		name = uuid.NewString()
	}

	pool := rxtspot.OnDemandNodePool{
		Name:              name,
		Org:               org,
		Cloudspace:        params.Cloudspace,
		ServerClass:       params.ServerClass,
		Desired:           params.Desired,
		CustomLabels:      params.CustomLabels,
		CustomAnnotations: params.CustomAnnotations,
	}

	if err := appCtx.Client.GetAPI().CreateOnDemandNodePool(ctx, org, pool); err != nil {
		return nil, err
	}
	return appCtx.Client.GetAPI().GetOnDemandNodePool(ctx, org, name)
}

func OnDemandUpdate(ctx context.Context, appCtx *app.Context, params OnDemandUpdateParams) (any, error) {
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return nil, fmt.Errorf("organization not specified and not configured")
	}
	if params.Name == "" || params.Cloudspace == "" {
		return nil, fmt.Errorf("name and cloudspace are required")
	}

	current, err := appCtx.Client.GetAPI().GetOnDemandNodePool(ctx, org, params.Name)
	if err != nil {
		return nil, err
	}

	updated := rxtspot.OnDemandNodePool{
		Name:       current.Name,
		Org:        org,
		Cloudspace: params.Cloudspace,
		// Keep serverclass from current; CLI doesn't allow changing it.
		ServerClass:       current.ServerClass,
		Desired:           current.Desired,
		CustomLabels:      current.CustomLabels,
		CustomAnnotations: current.CustomAnnotations,
	}

	if params.Desired != nil {
		updated.Desired = *params.Desired
	}
	if params.CustomLabels != nil {
		updated.CustomLabels = params.CustomLabels
	}
	if params.CustomAnnotations != nil {
		updated.CustomAnnotations = params.CustomAnnotations
	}

	if err := appCtx.Client.GetAPI().UpdateOnDemandNodePool(ctx, org, updated); err != nil {
		return nil, err
	}
	return appCtx.Client.GetAPI().GetOnDemandNodePool(ctx, org, params.Name)
}

func OnDemandDelete(ctx context.Context, appCtx *app.Context, params OnDemandDeleteParams) error {
	org := params.Org
	if org == "" {
		org = appCtx.Org
	}
	if org == "" {
		return fmt.Errorf("organization not specified and not configured")
	}
	if params.Name == "" {
		return fmt.Errorf("name is required")
	}
	return appCtx.Client.GetAPI().DeleteOnDemandNodePool(ctx, org, params.Name)
}

