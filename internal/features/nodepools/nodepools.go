package nodepools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	rxtspot "github.com/rackspace-spot/spot-go-sdk/api/v1"
	"github.com/rackspace-spot/spotctl/internal/app"
	"github.com/rackspace-spot/spotctl/internal/validation"
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

	AutoscalingEnabled  *bool `json:"autoscaling_enabled,omitempty" jsonschema:"Enable autoscaling; defaults to disabled"`
	AutoscalingMinNodes *int  `json:"autoscaling_min_nodes,omitempty" jsonschema:"Minimum number of nodes for autoscaling"`
	AutoscalingMaxNodes *int  `json:"autoscaling_max_nodes,omitempty" jsonschema:"Maximum number of nodes for autoscaling"`
}

type SpotUpdateParams struct {
	Org                 string            `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name                string            `json:"name" jsonschema:"Spot node pool name (UUID)"`
	Cloudspace          string            `json:"cloudspace" jsonschema:"Cloudspace name"`
	Desired             *int              `json:"desired,omitempty" jsonschema:"Desired number of nodes; if omitted, unchanged"`
	BidPrice            *string           `json:"bidprice,omitempty" jsonschema:"Maximum bid price; if omitted, unchanged"`
	AutoscalingEnabled  *bool             `json:"autoscaling_enabled,omitempty" jsonschema:"Enable or disable autoscaling; if omitted, unchanged"`
	AutoscalingMinNodes *int              `json:"autoscaling_min_nodes,omitempty" jsonschema:"Minimum number of nodes for autoscaling; if omitted, unchanged"`
	AutoscalingMaxNodes *int              `json:"autoscaling_max_nodes,omitempty" jsonschema:"Maximum number of nodes for autoscaling; if omitted, unchanged"`
	CustomLabels        map[string]string `json:"customLabels,omitempty" jsonschema:"Custom labels for the node pool; if omitted, unchanged"`
	CustomAnnotations   map[string]string `json:"customAnnotations,omitempty" jsonschema:"Custom annotations for the node pool; if omitted, unchanged"`
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

	// Validate bid price (B-01: strip $ and validate format)
	cleanBidPrice, err := validation.ValidateBidPrice(params.BidPrice)
	if err != nil {
		return nil, fmt.Errorf("invalid bid price: %w", err)
	}

	// Validate server class is available for spot bidding (B-02: check for deprecated BM classes)
	if err := validation.ValidateServerClassForSpotBidding(ctx, appCtx, params.ServerClass); err != nil {
		return nil, err
	}

	name := params.Name
	if name == "" {
		name = uuid.NewString()
	}

	// Autoscaling is required by the API on create. Default to a fixed-size
	// pool (disabled, min/max 0) unless flags request otherwise.
	if params.AutoscalingMinNodes != nil && params.AutoscalingMaxNodes != nil &&
		*params.AutoscalingMinNodes > *params.AutoscalingMaxNodes {
		return nil, fmt.Errorf("autoscaling min nodes (%d) cannot be greater than max nodes (%d)", *params.AutoscalingMinNodes, *params.AutoscalingMaxNodes)
	}
	autoscaling := rxtspot.Autoscaling{}
	if params.AutoscalingEnabled != nil {
		autoscaling.Enabled = *params.AutoscalingEnabled
	}
	if params.AutoscalingMinNodes != nil {
		autoscaling.MinNodes = int64(*params.AutoscalingMinNodes)
	}
	if params.AutoscalingMaxNodes != nil {
		autoscaling.MaxNodes = int64(*params.AutoscalingMaxNodes)
	}

	pool := rxtspot.SpotNodePool{
		Name:              name,
		Org:               org,
		Cloudspace:        params.Cloudspace,
		ServerClass:       params.ServerClass,
		Desired:           params.Desired,
		BidPrice:          cleanBidPrice,
		CustomLabels:      params.CustomLabels,
		CustomAnnotations: params.CustomAnnotations,
		Autoscaling:       &autoscaling,
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

	// Validate bid price if being updated (B-01: strip $ and validate format)
	var cleanBidPrice string
	if params.BidPrice != nil {
		var err error
		cleanBidPrice, err = validation.ValidateBidPrice(*params.BidPrice)
		if err != nil {
			return nil, fmt.Errorf("invalid bid price: %w", err)
		}
	} else {
		// Strip $ from current value if not updating (B-01)
		var err error
		cleanBidPrice, err = validation.ValidateBidPrice(current.BidPrice)
		if err != nil {
			return nil, fmt.Errorf("invalid current bid price: %w", err)
		}
	}

	// Build update options; current values are preserved as defaults.
	opts := rxtspot.SpotNodePoolUpdateOptions{
		Name:              params.Name,
		Desired:           params.Desired,
		BidPrice:          cleanBidPrice,
		CustomLabels:      params.CustomLabels,
		CustomAnnotations: params.CustomAnnotations,
	}

	// Validate autoscaling parameters
	if params.AutoscalingMinNodes != nil && params.AutoscalingMaxNodes != nil {
		if *params.AutoscalingMinNodes < 0 || *params.AutoscalingMaxNodes < 0 {
			return nil, fmt.Errorf("autoscaling min and max nodes must be non-negative")
		}
		if *params.AutoscalingMinNodes > *params.AutoscalingMaxNodes {
			return nil, fmt.Errorf("autoscaling min nodes (%d) cannot be greater than max nodes (%d)", *params.AutoscalingMinNodes, *params.AutoscalingMaxNodes)
		}
	}
	if params.AutoscalingEnabled != nil || params.AutoscalingMinNodes != nil || params.AutoscalingMaxNodes != nil {
		autoscaling := rxtspot.Autoscaling{}
		if current.Autoscaling != nil {
			autoscaling = *current.Autoscaling
		}
		if params.AutoscalingEnabled != nil {
			autoscaling.Enabled = *params.AutoscalingEnabled
		}
		if params.AutoscalingMinNodes != nil {
			autoscaling.MinNodes = int64(*params.AutoscalingMinNodes)
		}
		if params.AutoscalingMaxNodes != nil {
			autoscaling.MaxNodes = int64(*params.AutoscalingMaxNodes)
		}
		opts.Autoscaling = &autoscaling
	}

	if err := appCtx.Client.GetAPI().UpdateSpotNodePool(ctx, org, opts); err != nil {
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
	Org                 string            `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name                string            `json:"name" jsonschema:"On-demand node pool name (UUID)"`
	Cloudspace          string            `json:"cloudspace" jsonschema:"Cloudspace name"`
	Desired             *int              `json:"desired,omitempty" jsonschema:"Desired number of nodes; if omitted, unchanged"`
	AutoscalingEnabled  *bool             `json:"autoscaling_enabled,omitempty" jsonschema:"Enable or disable autoscaling; if omitted, unchanged"`
	AutoscalingMinNodes *int              `json:"autoscaling_min_nodes,omitempty" jsonschema:"Minimum number of nodes for autoscaling; if omitted, unchanged"`
	AutoscalingMaxNodes *int              `json:"autoscaling_max_nodes,omitempty" jsonschema:"Maximum number of nodes for autoscaling; if omitted, unchanged"`
	CustomLabels        map[string]string `json:"customLabels,omitempty" jsonschema:"Custom labels for the node pool; if omitted, unchanged"`
	CustomAnnotations   map[string]string `json:"customAnnotations,omitempty" jsonschema:"Custom annotations for the node pool; if omitted, unchanged"`
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

	// Build update options; current values are preserved as defaults.
	opts := rxtspot.OnDemandNodePoolUpdateOptions{
		Name:              params.Name,
		Desired:           params.Desired,
		CustomLabels:      params.CustomLabels,
		CustomAnnotations: params.CustomAnnotations,
	}

	// Validate autoscaling parameters
	if params.AutoscalingMinNodes != nil && params.AutoscalingMaxNodes != nil {
		if *params.AutoscalingMinNodes < 0 || *params.AutoscalingMaxNodes < 0 {
			return nil, fmt.Errorf("autoscaling min and max nodes must be non-negative")
		}
		if *params.AutoscalingMinNodes > *params.AutoscalingMaxNodes {
			return nil, fmt.Errorf("autoscaling min nodes (%d) cannot be greater than max nodes (%d)", *params.AutoscalingMinNodes, *params.AutoscalingMaxNodes)
		}
	}
	if params.AutoscalingEnabled != nil || params.AutoscalingMinNodes != nil || params.AutoscalingMaxNodes != nil {
		autoscaling := rxtspot.Autoscaling{}
		if current.Autoscaling != nil {
			autoscaling = *current.Autoscaling
		}
		if params.AutoscalingEnabled != nil {
			autoscaling.Enabled = *params.AutoscalingEnabled
		}
		if params.AutoscalingMinNodes != nil {
			autoscaling.MinNodes = int64(*params.AutoscalingMinNodes)
		}
		if params.AutoscalingMaxNodes != nil {
			autoscaling.MaxNodes = int64(*params.AutoscalingMaxNodes)
		}
		opts.Autoscaling = &autoscaling
	}

	if err := appCtx.Client.GetAPI().UpdateOnDemandNodePool(ctx, org, opts); err != nil {
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
