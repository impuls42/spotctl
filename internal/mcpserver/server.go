package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rackspace-spot/spotctl/internal/app"
	"github.com/rackspace-spot/spotctl/internal/features/cloudspaces"
	"github.com/rackspace-spot/spotctl/internal/features/nodepools"
	"github.com/rackspace-spot/spotctl/internal/features/organizations"
	"github.com/rackspace-spot/spotctl/internal/features/pricing"
	"github.com/rackspace-spot/spotctl/internal/features/regions"
	"github.com/rackspace-spot/spotctl/internal/features/serverclasses"
	"github.com/rackspace-spot/spotctl/internal/version"
)

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

	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
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

type CloudspacesDeleteParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"Cloudspace name to delete"`
}

type CloudspacesUpdateParams struct {
	Org                  string  `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name                 string  `json:"name" jsonschema:"Cloudspace name"`
	KubernetesVersion    *string `json:"kubernetes_version,omitempty" jsonschema:"Kubernetes version; if omitted, unchanged"`
	HAControlPlane       *bool   `json:"ha_control_plane,omitempty" jsonschema:"Enable or disable HA control plane; if omitted, unchanged"`
	PreemptionWebhookURL *string `json:"preemption_webhook_url,omitempty" jsonschema:"Preemption webhook URL; if omitted, unchanged"`
	CNI                  *string `json:"cni,omitempty" jsonschema:"Container Network Interface (CNI) plugin; if omitted, unchanged"`
}

type CloudspacesGetConfigParams struct {
	Org  string `json:"org,omitempty" jsonschema:"Organization ID; falls back to configured org if empty"`
	Name string `json:"name" jsonschema:"Cloudspace name"`
}

func registerCloudspacesTools(server *mcp.Server) {
	// list
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cloudspaces_list",
		Description: "List Rackspace Spot cloudspaces in an organization",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *CloudspacesListParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := cloudspaces.List(ctx, appCtx, appCtx.Org)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// get
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cloudspaces_get",
		Description: "Get details of a specific Rackspace Spot cloudspace",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *CloudspacesGetParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := cloudspaces.Get(ctx, appCtx, appCtx.Org, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// create
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cloudspaces_create",
		Description: "Create a new Rackspace Spot cloudspace with optional node pools",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *cloudspaces.CreateParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, Region: params.Region, RequireOrg: true, RequireRegion: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := cloudspaces.Create(ctx, appCtx, *params)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// update (T-04)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cloudspaces_update",
		Description: "Update an existing Rackspace Spot cloudspace (kubernetes version, HA control plane, CNI, preemption webhook)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *CloudspacesUpdateParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		// Convert CloudspacesUpdateParams to cloudspaces.UpdateParams
		updateParams := cloudspaces.UpdateParams{
			Name:                 params.Name,
			KubernetesVersion:    params.KubernetesVersion,
			HAControlPlane:       params.HAControlPlane,
			PreemptionWebhookURL: params.PreemptionWebhookURL,
			CNI:                  params.CNI,
		}
		res, err := cloudspaces.Update(ctx, appCtx, appCtx.Org, updateParams)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// delete
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cloudspaces_delete",
		Description: "Delete a Rackspace Spot cloudspace",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *CloudspacesDeleteParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		if err := cloudspaces.Delete(ctx, appCtx, appCtx.Org, params.Name); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("cloudspace %q deleted", params.Name))
	})

	// get-config
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cloudspaces_get_config",
		Description: "Get kubeconfig for a Rackspace Spot cloudspace",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *CloudspacesGetConfigParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		cfgText, err := cloudspaces.GetConfig(ctx, appCtx, appCtx.Org, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return textResult(cfgText)
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
		appCtx, err := app.Load(ctx, app.LoadOptions{})
		if err != nil {
			return nil, nil, err
		}
		res, err := regions.List(ctx, appCtx)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "regions_get",
		Description: "Get details of a specific Rackspace Spot region",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *RegionsGetParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		appCtx, err := app.Load(ctx, app.LoadOptions{})
		if err != nil {
			return nil, nil, err
		}
		res, err := regions.Get(ctx, appCtx, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
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
		appCtx, err := app.Load(ctx, app.LoadOptions{})
		if err != nil {
			return nil, nil, err
		}
		res, err := organizations.List(ctx, appCtx)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "organizations_get",
		Description: "Get details for a specific organization by name",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *OrganizationsGetParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		appCtx, err := app.Load(ctx, app.LoadOptions{})
		if err != nil {
			return nil, nil, err
		}
		res, err := organizations.GetByName(ctx, appCtx, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
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
		appCtx, err := app.Load(ctx, app.LoadOptions{Region: params.Region, RequireRegion: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := serverclasses.List(ctx, appCtx, appCtx.Region)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "serverclasses_get",
		Description: "Get details of a specific server class",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *ServerclassesGetParams) (*mcp.CallToolResult, any, error) {
		if params.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		appCtx, err := app.Load(ctx, app.LoadOptions{})
		if err != nil {
			return nil, nil, err
		}
		res, err := serverclasses.Get(ctx, appCtx, params.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})
}

// -----------------------
// Pricing tools
// -----------------------

type PricingGetParams struct {
	ServerClass string `json:"serverclass" jsonschema:"Server class name to get pricing for"`
}

type PricingGetAllParams struct{}

type PricingGetHistoryParams struct {
	ServerClass string `json:"serverclass" jsonschema:"Server class name to get price history for"`
}

type PricingGetPercentilesParams struct {
	Region      string `json:"region,omitempty" jsonschema:"Region to filter percentiles by (optional)"`
	ServerClass string `json:"serverclass,omitempty" jsonschema:"Server class to filter percentiles by (optional)"`
}

type PricingGetComparableParams struct {
	Region      string `json:"region,omitempty" jsonschema:"Region to filter comparable prices by (optional)"`
	ServerClass string `json:"serverclass,omitempty" jsonschema:"Server class to filter comparable prices by (optional)"`
}

func registerPricingTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "pricing_get",
		Description: "Get current market price details for a server class",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *PricingGetParams) (*mcp.CallToolResult, any, error) {
		if params.ServerClass == "" {
			return nil, nil, fmt.Errorf("serverclass is required")
		}
		appCtx, err := app.Load(ctx, app.LoadOptions{})
		if err != nil {
			return nil, nil, err
		}
		res, err := pricing.GetForServerClass(ctx, appCtx, params.ServerClass)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pricing_get_all",
		Description: "Get pricing details for all server classes",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *PricingGetAllParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{})
		if err != nil {
			return nil, nil, err
		}
		res, err := pricing.GetAll(ctx, appCtx)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// Pricing get history (T-02)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "pricing_get_history",
		Description: "Get historical price data for a server class (enables volatility analysis and P95 bid recommendations)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *PricingGetHistoryParams) (*mcp.CallToolResult, any, error) {
		if params.ServerClass == "" {
			return nil, nil, fmt.Errorf("serverclass is required")
		}
		res, err := GetPriceHistory(ctx, params.ServerClass)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// Pricing get percentiles (T-03)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "pricing_get_percentiles",
		Description: "Get price percentile distributions for server classes (enables intelligent bid setting with P50/P95 values)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *PricingGetPercentilesParams) (*mcp.CallToolResult, any, error) {
		res, err := GetPricePercentiles(ctx, params.Region, params.ServerClass)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// Pricing get comparable (T-06)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "pricing_get_comparable",
		Description: "Get Rackspace Spot prices vs hyperscaler equivalents (AWS/GCP/Azure) for cost optimization analysis",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *PricingGetComparableParams) (*mcp.CallToolResult, any, error) {
		res, err := GetComparablePrices(ctx, params.Region, params.ServerClass)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})
}

func registerNodepoolsTools(server *mcp.Server) {
	// Spot list
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spot_nodepools_list",
		Description: "List spot node pools in a cloudspace",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *nodepools.SpotListParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := nodepools.SpotList(ctx, appCtx, *params)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// Spot get
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spot_nodepools_get",
		Description: "Get a specific spot node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *nodepools.SpotGetParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := nodepools.SpotGet(ctx, appCtx, *params)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// Spot create
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spot_nodepools_create",
		Description: "Create a new spot node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *nodepools.SpotCreateParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := nodepools.SpotCreate(ctx, appCtx, *params)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// Spot update
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spot_nodepools_update",
		Description: "Update an existing spot node pool (bid price, desired count, autoscaling, labels,annotations)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *nodepools.SpotUpdateParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := nodepools.SpotUpdate(ctx, appCtx, *params)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// Spot delete
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spot_nodepools_delete",
		Description: "Delete a spot node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *nodepools.SpotDeleteParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		if err := nodepools.SpotDelete(ctx, appCtx, *params); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("spot node pool %q deleted", params.Name))
	})

	// On-demand list
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ondemand_nodepools_list",
		Description: "List on-demand node pools in a cloudspace",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *nodepools.OnDemandListParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := nodepools.OnDemandList(ctx, appCtx, *params)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// On-demand get
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ondemand_nodepools_get",
		Description: "Get a specific on-demand node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *nodepools.OnDemandGetParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := nodepools.OnDemandGet(ctx, appCtx, *params)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// On-demand create
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ondemand_nodepools_create",
		Description: "Create a new on-demand node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *nodepools.OnDemandCreateParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := nodepools.OnDemandCreate(ctx, appCtx, *params)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// On-demand update
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ondemand_nodepools_update",
		Description: "Update an existing on-demand node pool (desired count, autoscaling, labels, annotations)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *nodepools.OnDemandUpdateParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		res, err := nodepools.OnDemandUpdate(ctx, appCtx, *params)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(res)
	})

	// On-demand delete
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ondemand_nodepools_delete",
		Description: "Delete an on-demand node pool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params *nodepools.OnDemandDeleteParams) (*mcp.CallToolResult, any, error) {
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: params.Org, RequireOrg: true})
		if err != nil {
			return nil, nil, err
		}
		if err := nodepools.OnDemandDelete(ctx, appCtx, *params); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("on-demand node pool %q deleted", params.Name))
	})
}
