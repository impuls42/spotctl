package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	rxtspot "github.com/rackspace-spot/spot-go-sdk/api/v1"
	"github.com/rackspace-spot/spotctl/internal"
	config "github.com/rackspace-spot/spotctl/pkg"
	"github.com/rackspace-spot/spotctl/internal/version"
)

// helper to load CLI config or return a clear error
func loadConfig() (*config.SpotConfig, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load spotctl config, please run 'spotctl configure' first: %w", err)
	}
	return cfg, nil
}

// helper to create an authenticated client from SpotConfig
func newClientFromConfig(cfg *config.SpotConfig) (*internal.Client, error) {
	return internal.NewClientWithTokens(cfg.RefreshToken, cfg.AccessToken)
}

// NewServer constructs the MCP server with all tools registered.
func NewServer() *mcp.Server {
	impl := &mcp.Implementation{
		Name:    "spotctl-mcp",
		Version: version.GetVersion(),
	}

	server := mcp.NewServer(impl, nil)

	// Register tools grouped by feature area.
	registerCloudspacesTools(server)
	registerRegionsTools(server)
	registerOrganizationsTools(server)
	registerServerclassesTools(server)
	registerPricingTools(server)
	registerNodepoolsTools(server)

	return server
}

// RunStdio runs the MCP server over stdio until the client disconnects.
func RunStdio(ctx context.Context) error {
	server := NewServer()
	return server.Run(ctx, &mcp.StdioTransport{})
}

// RunHTTP runs the MCP server over the streamable HTTP transport on the given addr.
func RunHTTP(ctx context.Context, addr string) error {
	server := NewServer()

	handler := mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server {
			return server
		},
		nil,
	)

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Shutdown on ctx cancel.
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	return srv.ListenAndServe()
}

// jsonResult wraps arbitrary Go values into a JSON text content result.
func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}

// textResult wraps plain text into a tool result.
func textResult(text string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil, nil
}

// -----------------------
// Cloudspaces tools
// -----------------------

type CloudspacesListParams struct {
	Org string `json:"org,omitempty" jsonschema:"Organization ID to list cloudspaces for; falls back to configured org if empty"`
}

type CloudspacesGetParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"Cloudspace name"`
}

type CloudspacesCreateParams struct {
	Name                 string                           `json:"name" jsonschema:"Cloudspace name"`
	Org                  string                           `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Region               string                           `json:"region,omitempty" jsonschema:"Region; falls back to configured region if empty"`
	KubernetesVersion    string                           `json:"kubernetesVersion,omitempty" jsonschema:"Kubernetes version (e.g. 1.31.1); default from CLI if empty"`
	PreemptionWebhookURL string                           `json:"preemptionWebhookURL,omitempty" jsonschema:"Preemption webhook URL"`
	CNI                  string                           `json:"cni,omitempty" jsonschema:"CNI plugin (e.g. calico, cilium)"`
	SpotNodePools        []rxtspot.SpotNodePool           `json:"spotNodePools,omitempty" jsonschema:"Spot node pools to create in this cloudspace"`
	OnDemandNodePools    []rxtspot.OnDemandNodePool       `json:"onDemandNodePools,omitempty" jsonschema:"On-demand node pools to create in this cloudspace"`
}

type CloudspacesDeleteParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"Cloudspace name to delete"`
}

func registerCloudspacesTools(server *mcp.Server) {
	// list
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cloudspaces_list",
		Description: "List Rackspace Spot cloudspaces in an organization",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *CloudspacesListParams) (*mcp.CallToolResult, any, error) {
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}

		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		cloudspaces, err := client.GetAPI().ListCloudspaces(ctx, org)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(cloudspaces)
	})

	// get
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cloudspaces_get",
		Description: "Get details of a specific Rackspace Spot cloudspace",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *CloudspacesGetParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}

		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		cloudspace, err := client.GetAPI().GetCloudspace(ctx, org, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(cloudspace)
	})

	// create
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cloudspaces_create",
		Description: "Create a new Rackspace Spot cloudspace with optional node pools",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *CloudspacesCreateParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		region := params.Region
		if region == "" {
			region = cfg.Region
		}
		if region == "" {
			return nil, nil, fmt.Errorf("region not specified and not configured; run 'spotctl configure' or pass region")
		}

		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}

		cloudspace := rxtspot.CloudSpace{
			Name:                 params.Name,
			Org:                  org,
			Region:               region,
			KubernetesVersion:    params.KubernetesVersion,
			CNI:                  params.CNI,
			PreemptionWebhookURL: params.PreemptionWebhookURL,
		}

		if err := client.GetAPI().CreateCloudspace(ctx, cloudspace); err != nil {
			return nil, nil, err
		}

		// Create spot node pools if any
		for _, pool := range params.SpotNodePools {
			p := rxtspot.SpotNodePool{
				Name:        pool.Name,
				Org:         org,
				Cloudspace:  params.Name,
				ServerClass: pool.ServerClass,
				BidPrice:    pool.BidPrice,
				Desired:     pool.Desired,
			}
			if err := client.GetAPI().CreateSpotNodePool(ctx, org, p); err != nil {
				return nil, nil, fmt.Errorf("failed creating spot node pool %s: %w", p.Name, err)
			}
		}

		// Create on-demand node pools if any
		for _, pool := range params.OnDemandNodePools {
			p := rxtspot.OnDemandNodePool{
				Name:        pool.Name,
				Org:         org,
				Cloudspace:  params.Name,
				ServerClass: pool.ServerClass,
				Desired:     pool.Desired,
			}
			if err := client.GetAPI().CreateOnDemandNodePool(ctx, org, p); err != nil {
				return nil, nil, fmt.Errorf("failed creating on-demand node pool %s: %w", p.Name, err)
			}
		}

		created, err := client.GetAPI().GetCloudspace(ctx, org, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(created)
	})

	// delete
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cloudspaces_delete",
		Description: "Delete a Rackspace Spot cloudspace",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *CloudspacesDeleteParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		if err := client.GetAPI().DeleteCloudspace(ctx, org, params.Name); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("cloudspace %q deleted", params.Name))
	})
}

// -----------------------
// Regions tools
// -----------------------

type RegionsListParams struct{}

type RegionsGetParams struct {
	Name string `json:"name" jsonschema:"Region name"`
}

func registerRegionsTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "regions_list",
		Description: "List Rackspace Spot regions",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *RegionsListParams) (*mcp.CallToolResult, any, error) {
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		regions, err := client.GetAPI().ListRegions(ctx)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(regions)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "regions_get",
		Description: "Get details of a specific Rackspace Spot region",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *RegionsGetParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		region, err := client.GetAPI().GetRegion(ctx, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(region)
	})
}

// -----------------------
// Organizations tools
// -----------------------

type OrganizationsListParams struct{}

type OrganizationsGetParams struct {
	Name string `json:"name" jsonschema:"Organization name"`
}

func registerOrganizationsTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "organizations_list",
		Description: "List organizations accessible by the authenticated user",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *OrganizationsListParams) (*mcp.CallToolResult, any, error) {
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		orgs, err := client.GetAPI().ListOrganizations(ctx)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(orgs)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "organizations_get",
		Description: "Get details for a specific organization by name",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *OrganizationsGetParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		orgs, err := client.GetAPI().ListOrganizations(ctx)
		if err != nil {
			return nil, nil, err
		}
		for _, org := range orgs {
			if org.Name == params.Name {
				return jsonResult(org)
			}
		}
		return nil, nil, fmt.Errorf("organization %q not found", params.Name)
	})
}

// -----------------------
// Serverclasses tools
// -----------------------

type ServerclassesListParams struct {
	Region string `json:"region,omitempty" jsonschema:"Region to list server classes for; falls back to configured region if empty"`
}

type ServerclassesGetParams struct {
	Name string `json:"name" jsonschema:"Server class name"`
}

func registerServerclassesTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "serverclasses_list",
		Description: "List server classes for a region",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *ServerclassesListParams) (*mcp.CallToolResult, any, error) {
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		region := params.Region
		if region == "" {
			region = cfg.Region
		}
		if region == "" {
			return nil, nil, fmt.Errorf("region not specified and not configured; run 'spotctl configure' or pass region")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		serverclasses, err := client.GetAPI().ListServerClasses(ctx, region)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(serverclasses)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "serverclasses_get",
		Description: "Get details of a specific server class",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *ServerclassesGetParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		serverclass, err := client.GetAPI().GetServerClass(ctx, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(serverclass)
	})
}

// -----------------------
// Pricing tools
// -----------------------

type PricingGetParams struct {
	ServerClass string `json:"serverclass" jsonschema:"Server class name to get pricing for"`
}

type PricingGetAllParams struct{}

func registerPricingTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "pricing_get",
		Description: "Get current market price details for a server class",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *PricingGetParams) (*mcp.CallToolResult, any, error) {
		if params.ServerClass == "" {
			return nil, nil, fmt.Errorf("serverclass is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		pricing, err := client.GetAPI().GetPriceDetailsForServerClass(ctx, params.ServerClass)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(pricing)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pricing_get_all",
		Description: "Get pricing details for all server classes",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *PricingGetAllParams) (*mcp.CallToolResult, any, error) {
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		pricing, err := client.GetAPI().GetPriceDetails(ctx)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(pricing)
	})
}

// -----------------------
// Nodepools tools
// -----------------------

type SpotNodepoolsListParams struct {
	Org        string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Cloudspace string `json:"cloudspace" jsonschema:"Cloudspace name"`
}

type SpotNodepoolsGetParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"Spot node pool name (UUID)"`
}

type SpotNodepoolsCreateParams struct {
	Org               string            `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Cloudspace        string            `json:"cloudspace" jsonschema:"Cloudspace name"`
	ServerClass       string            `json:"serverclass" jsonschema:"Server class"`
	Desired           int               `json:"desired" jsonschema:"Desired number of nodes"`
	BidPrice          string            `json:"bidprice" jsonschema:"Maximum bid price"`
	CustomLabels      map[string]string `json:"customLabels,omitempty" jsonschema:"Custom labels for the node pool"`
	CustomAnnotations map[string]string `json:"customAnnotations,omitempty" jsonschema:"Custom annotations for the node pool"`
	Name              string            `json:"name,omitempty" jsonschema:"Optional explicit node pool name; generated if empty"`
}

type SpotNodepoolsUpdateParams struct {
	Org               string            `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name              string            `json:"name" jsonschema:"Spot node pool name (UUID)"`
	Cloudspace        string            `json:"cloudspace" jsonschema:"Cloudspace name"`
	Desired           *int              `json:"desired,omitempty" jsonschema:"Desired number of nodes; if omitted, unchanged"`
	BidPrice          string            `json:"bidprice,omitempty" jsonschema:"Maximum bid price; if empty, unchanged"`
	CustomLabels      map[string]string `json:"customLabels,omitempty" jsonschema:"Custom labels for the node pool"`
	CustomAnnotations map[string]string `json:"customAnnotations,omitempty" jsonschema:"Custom annotations for the node pool"`
}

type SpotNodepoolsDeleteParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"Spot node pool name (UUID)"`
}

type OndemandNodepoolsListParams struct {
	Org        string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Cloudspace string `json:"cloudspace" jsonschema:"Cloudspace name"`
}

type OndemandNodepoolsGetParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"On-demand node pool name (UUID)"`
}

type OndemandNodepoolsCreateParams struct {
	Org               string            `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Cloudspace        string            `json:"cloudspace" jsonschema:"Cloudspace name"`
	ServerClass       string            `json:"serverclass" jsonschema:"Server class"`
	Desired           int               `json:"desired" jsonschema:"Desired number of nodes"`
	CustomLabels      map[string]string `json:"customLabels,omitempty" jsonschema:"Custom labels for the node pool"`
	CustomAnnotations map[string]string `json:"customAnnotations,omitempty" jsonschema:"Custom annotations for the node pool"`
	Name              string            `json:"name,omitempty" jsonschema:"Optional explicit node pool name; generated if empty"`
}

type OndemandNodepoolsUpdateParams struct {
	Org               string            `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name              string            `json:"name" jsonschema:"On-demand node pool name (UUID)"`
	Cloudspace        string            `json:"cloudspace" jsonschema:"Cloudspace name"`
	Desired           *int              `json:"desired,omitempty" jsonschema:"Desired number of nodes; if omitted, unchanged"`
	CustomLabels      map[string]string `json:"customLabels,omitempty" jsonschema:"Custom labels for the node pool"`
	CustomAnnotations map[string]string `json:"customAnnotations,omitempty" jsonschema:"Custom annotations for the node pool"`
}

type OndemandNodepoolsDeleteParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"On-demand node pool name (UUID)"`
}

func registerNodepoolsTools(server *mcp.Server) {
	// Spot list
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spot_nodepools_list",
		Description: "List spot node pools in a cloudspace",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *SpotNodepoolsListParams) (*mcp.CallToolResult, any, error) {
		if params.Cloudspace == "" {
			return nil, nil, fmt.Errorf("cloudspace is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		pools, err := client.GetAPI().ListSpotNodePools(ctx, org, params.Cloudspace)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(pools)
	})

	// Spot get
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spot_nodepools_get",
		Description: "Get a specific spot node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *SpotNodepoolsGetParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		pool, err := client.GetAPI().GetSpotNodePool(ctx, org, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(pool)
	})

	// Spot create
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spot_nodepools_create",
		Description: "Create a new spot node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *SpotNodepoolsCreateParams) (*mcp.CallToolResult, any, error) {
		if params.Cloudspace == "" || params.ServerClass == "" || params.Desired <= 0 || params.BidPrice == "" {
			return nil, nil, fmt.Errorf("cloudspace, serverclass, desired (>0), and bidprice are required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}

		pool := &rxtspot.SpotNodePool{
			Name:              params.Name,
			Org:               org,
			Cloudspace:        params.Cloudspace,
			ServerClass:       params.ServerClass,
			Desired:           params.Desired,
			BidPrice:          params.BidPrice,
			CustomLabels:      params.CustomLabels,
			CustomAnnotations: params.CustomAnnotations,
		}

		if err := client.GetAPI().CreateSpotNodePool(ctx, org, *pool); err != nil {
			return nil, nil, err
		}
		created, err := client.GetAPI().GetSpotNodePool(ctx, org, pool.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(created)
	})

	// Spot update
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spot_nodepools_update",
		Description: "Update an existing spot node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *SpotNodepoolsUpdateParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" || params.Cloudspace == "" {
			return nil, nil, fmt.Errorf("name and cloudspace are required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}

		desired := 0
		if params.Desired != nil {
			desired = *params.Desired
		}

		pool := &rxtspot.SpotNodePool{
			Name:              params.Name,
			Org:               org,
			Cloudspace:        params.Cloudspace,
			Desired:           desired,
			BidPrice:          params.BidPrice,
			CustomLabels:      params.CustomLabels,
			CustomAnnotations: params.CustomAnnotations,
		}

		if err := client.GetAPI().UpdateSpotNodePool(ctx, org, *pool); err != nil {
			return nil, nil, err
		}
		updated, err := client.GetAPI().GetSpotNodePool(ctx, org, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(updated)
	})

	// Spot delete
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spot_nodepools_delete",
		Description: "Delete a spot node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *SpotNodepoolsDeleteParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		if err := client.GetAPI().DeleteSpotNodePool(ctx, org, params.Name); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("spot node pool %q deleted", params.Name))
	})

	// On-demand list
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ondemand_nodepools_list",
		Description: "List on-demand node pools in a cloudspace",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *OndemandNodepoolsListParams) (*mcp.CallToolResult, any, error) {
		if params.Cloudspace == "" {
			return nil, nil, fmt.Errorf("cloudspace is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		pools, err := client.GetAPI().ListOnDemandNodePools(ctx, org, params.Cloudspace)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(pools)
	})

	// On-demand get
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ondemand_nodepools_get",
		Description: "Get a specific on-demand node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *OndemandNodepoolsGetParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		pool, err := client.GetAPI().GetOnDemandNodePool(ctx, org, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(pool)
	})

	// On-demand create
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ondemand_nodepools_create",
		Description: "Create a new on-demand node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *OndemandNodepoolsCreateParams) (*mcp.CallToolResult, any, error) {
		if params.Cloudspace == "" || params.ServerClass == "" || params.Desired <= 0 {
			return nil, nil, fmt.Errorf("cloudspace, serverclass, and desired (>0) are required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}

		pool := &rxtspot.OnDemandNodePool{
			Name:              params.Name,
			Org:               org,
			Cloudspace:        params.Cloudspace,
			ServerClass:       params.ServerClass,
			Desired:           params.Desired,
			CustomLabels:      params.CustomLabels,
			CustomAnnotations: params.CustomAnnotations,
		}

		if err := client.GetAPI().CreateOnDemandNodePool(ctx, org, *pool); err != nil {
			return nil, nil, err
		}
		created, err := client.GetAPI().GetOnDemandNodePool(ctx, org, pool.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(created)
	})

	// On-demand update
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ondemand_nodepools_update",
		Description: "Update an existing on-demand node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *OndemandNodepoolsUpdateParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" || params.Cloudspace == "" {
			return nil, nil, fmt.Errorf("name and cloudspace are required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}

		desired := 0
		if params.Desired != nil {
			desired = *params.Desired
		}

		pool := &rxtspot.OnDemandNodePool{
			Name:              params.Name,
			Org:               org,
			Cloudspace:        params.Cloudspace,
			Desired:           desired,
			CustomLabels:      params.CustomLabels,
			CustomAnnotations: params.CustomAnnotations,
		}

		if err := client.GetAPI().UpdateOnDemandNodePool(ctx, org, *pool); err != nil {
			return nil, nil, err
		}
		updated, err := client.GetAPI().GetOnDemandNodePool(ctx, org, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(updated)
	})

	// On-demand delete
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ondemand_nodepools_delete",
		Description: "Delete an on-demand node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *OndemandNodepoolsDeleteParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		cfg, err := loadConfig()
		if err != nil {
			return nil, nil, err
		}
		org := params.Org
		if org == "" {
			org = cfg.Org
		}
		if org == "" {
			return nil, nil, fmt.Errorf("organization not specified and not configured; run 'spotctl configure' or pass org")
		}
		client, err := newClientFromConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
		if err := client.GetAPI().DeleteOnDemandNodePool(ctx, org, params.Name); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("on-demand node pool %q deleted", params.Name))
	})
}

