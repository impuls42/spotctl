package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/google/uuid"
	rxtspot "github.com/rackspace-spot/spot-go-sdk/api/v1"
	"github.com/rackspace-spot/spotctl/internal"
	"github.com/rackspace-spot/spotctl/internal/app"
	featvmcs "github.com/rackspace-spot/spotctl/internal/features/vmcloudspaces"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// createVMCloudSpaceParams holds all parameters needed for VM cloudspace creation
type createVMCloudSpaceParams struct {
	Name        string              `json:"name" yaml:"name"`
	Org         string              `json:"org" yaml:"org"`
	Region      string              `json:"region" yaml:"region"`
	Webhook     string              `json:"webhook,omitempty" yaml:"webhook,omitempty"`
	VMSshKeyRef rxtspot.VMSshKeyRef `json:"vmSshKeyRef" yaml:"vmSshKeyRef"`
	ConfigPath  string              `json:"-" yaml:"-"`
	VMPools     []rxtspot.VMPool    `json:"vmPools,omitempty" yaml:"vmPools,omitempty"`
}

// vmCloudSpacesCmd represents the vmcloudspaces command
var vmCloudSpacesCmd = &cobra.Command{
	Use:     "vmcloudspaces",
	Short:   "Manage VM cloudspaces",
	Long:    `Manage Rackspace Spot VM cloudspaces.`,
	Aliases: []string{"vmcs", "vmcloudspace"},
}

func init() {
	rootCmd.AddCommand(vmCloudSpacesCmd)
	vmCloudSpacesCmd.AddCommand(vmcsListCmd)
	vmCloudSpacesCmd.AddCommand(vmcsCreateCmd)
	vmCloudSpacesCmd.AddCommand(vmcsGetCmd)
	vmCloudSpacesCmd.AddCommand(vmcsUpdateCmd)
	vmCloudSpacesCmd.AddCommand(vmcsDeleteCmd)

	// Flags for vmcloudspaces list
	vmcsListCmd.Flags().String("org", "", "Organization name")
	vmcsListCmd.Flags().StringP("output", "o", "json", "Output format (json, table, yaml)")

	// Flags for vmcloudspaces create
	vmcsCreateCmd.Flags().String("name", "", "VM cloudspace name")
	vmcsCreateCmd.Flags().String("org", "", "Organization name")
	vmcsCreateCmd.Flags().String("region", "", "Region")
	vmcsCreateCmd.Flags().String("vm-ssh-key-name", "", "VM SSH key name (required)")
	vmcsCreateCmd.Flags().String("vm-ssh-key-namespace", "", "VM SSH key namespace (optional, defaults to org namespace)")
	vmcsCreateCmd.Flags().String("webhook", "", "Preemption webhook URL")
	vmcsCreateCmd.Flags().StringArray("vm-pool", []string{}, `VM pool details in key=value format (e.g., serverclass=gp.vs1.medium-ord,bidprice=0.02,desired=5,pooltype=spot,vmimage=ubuntu24.04)`)
	vmcsCreateCmd.Flags().String("vm-userdata", "", "Cloud-init user data for VM pools (raw text or base64-encoded, optional)")
	vmcsCreateCmd.Flags().String("vm-userdata-from-script", "", "Path to cloud-init script file for VM pools (optional)")
	vmcsCreateCmd.Flags().String("config", "", "Path to config file (YAML or JSON)")

	// Flags for vmcloudspaces update
	vmcsUpdateCmd.Flags().String("name", "", "VM cloudspace name (required)")
	vmcsUpdateCmd.Flags().String("org", "", "Organization name")
	vmcsUpdateCmd.Flags().String("webhook", "", "Preemption webhook URL")
	vmcsUpdateCmd.MarkFlagRequired("name")

	// Flags for vmcloudspaces get
	vmcsGetCmd.Flags().String("name", "", "VM cloudspace name (required)")
	vmcsGetCmd.Flags().String("org", "", "Organization name")
	vmcsGetCmd.MarkFlagRequired("name")

	// Flags for vmcloudspaces delete
	vmcsDeleteCmd.Flags().String("name", "", "VM cloudspace name (required)")
	vmcsDeleteCmd.Flags().String("org", "", "Organization name")
	vmcsDeleteCmd.MarkFlagRequired("name")
	vmcsDeleteCmd.Flags().BoolP("yes", "y", false, "Automatic yes to prompts; assume \"yes\" as answer")
}

// vmcsListCmd lists all VM cloudspaces
var vmcsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List VM cloudspaces",
	Long:  `List all VM cloudspaces in an organization.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		org, _ := cmd.Flags().GetString("org")
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{Org: org, RequireOrg: true})
		if err != nil {
			return err
		}

		vmCloudSpaces, err := featvmcs.List(cmd.Context(), appCtx, appCtx.Org)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		return internal.OutputData(vmCloudSpaces, outputFormat)
	},
}

// vmcsCreateCmd creates a new VM cloudspace
var vmcsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new VM cloudspace",
	Long:  `Create a new Rackspace Spot VM cloudspace with optional VM pools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\n\nOperation cancelled by user")
			cancel()
		}()

		org, _ := cmd.Flags().GetString("org")
		region, _ := cmd.Flags().GetString("region")
		appCtx, err := app.Load(ctx, app.LoadOptions{Org: org, Region: region})
		if err != nil {
			return err
		}

		params, err := loadVMCSParamsFromFlags(cmd)
		if err != nil {
			return fmt.Errorf("failed to load parameters: %w", err)
		}

		// Fall back to resolved defaults (a config file passed via --config may
		// already have set these, which takes precedence).
		if params.Org == "" {
			params.Org = appCtx.Org
		}
		if params.Region == "" {
			params.Region = appCtx.Region
		}
		if params.Name == "" {
			return fmt.Errorf("name is required")
		}
		if params.Org == "" {
			return fmt.Errorf("Organization is required")
		}
		if params.Region == "" {
			return fmt.Errorf("region is required")
		}
		if params.VMSshKeyRef.Name == "" {
			return fmt.Errorf("vm-ssh-key-name is required")
		}

		vmUserData, _ := cmd.Flags().GetString("vm-userdata")
		vmUserDataFromScript, _ := cmd.Flags().GetString("vm-userdata-from-script")

		vmcsResult, err := featvmcs.Create(ctx, appCtx, featvmcs.CreateParams{
			Org:                params.Org,
			Name:               params.Name,
			Region:             params.Region,
			Webhook:            params.Webhook,
			VMSshKeyRef:        params.VMSshKeyRef,
			VMPools:            params.VMPools,
			UserData:           vmUserData,
			UserDataFromScript: vmUserDataFromScript,
		})
		if err != nil {
			return err
		}

		fmt.Printf("\n%s Successfully created VM cloudspace %s in region %s\n",
			color.GreenString("✓"),
			color.CyanString(vmcsResult.Name),
			color.CyanString(vmcsResult.Region),
		)

		return internal.OutputData(vmcsResult, outputFormat)
	},
}

// vmcsGetCmd gets details about a specific VM cloudspace
var vmcsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get VM cloudspace details",
	Long:  `Get details about a specific VM cloudspace.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("name is required")
		}

		org, _ := cmd.Flags().GetString("org")
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{Org: org, RequireOrg: true})
		if err != nil {
			return err
		}

		vmcs, err := featvmcs.Get(cmd.Context(), appCtx, appCtx.Org, name)
		if err != nil {
			if rxtspot.IsNotFound(err) {
				return fmt.Errorf("VM cloudspace '%s' not found", name)
			}
			return fmt.Errorf("failed to get VM cloudspace: %w", err)
		}

		outputFmt, _ := cmd.Flags().GetString("output")
		if outputFmt == "" {
			outputFmt = "json"
		}

		return internal.OutputData(vmcs, outputFmt)
	},
}

// vmcsDeleteCmd deletes a VM cloudspace
var vmcsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a VM cloudspace",
	Long:  `Delete a VM cloudspace and all its resources.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("name is required")
		}

		org, _ := cmd.Flags().GetString("org")
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{Org: org, RequireOrg: true})
		if err != nil {
			return err
		}

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			prompt := color.New(color.FgYellow).PrintfFunc()
			prompt("Are you sure you want to delete VM cloudspace '%s'? (y/N): ", name)

			var response string
			_, err := fmt.Scanln(&response)
			if err != nil || (response != "y" && response != "Y") {
				fmt.Println("Aborted.")
				return nil
			}
		}

		err = featvmcs.Delete(cmd.Context(), appCtx, appCtx.Org, name)
		if err != nil {
			if rxtspot.IsNotFound(err) {
				return fmt.Errorf("VM cloudspace '%s' not found", name)
			}
			if rxtspot.IsForbidden(err) {
				return fmt.Errorf("forbidden: %w", err)
			}
			if rxtspot.IsConflict(err) {
				return fmt.Errorf("conflict: %w", err)
			}
			return fmt.Errorf("%w", err)
		}

		fmt.Printf("VM cloudspace '%s' deleted successfully\n", name)
		return nil
	},
}

// vmcsUpdateCmd updates a VM cloudspace (currently only webhook can be modified)
var vmcsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a VM cloudspace",
	Long:  `Update an existing VM cloudspace. Currently only the webhook URL can be modified.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("name is required")
		}

		org, _ := cmd.Flags().GetString("org")
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{Org: org, RequireOrg: true})
		if err != nil {
			return err
		}

		webhook, _ := cmd.Flags().GetString("webhook")

		if err := featvmcs.Update(cmd.Context(), appCtx, appCtx.Org, name, webhook); err != nil {
			if rxtspot.IsNotFound(err) {
				return fmt.Errorf("VM cloudspace '%s' not found", name)
			}
			return fmt.Errorf("failed to update VM cloudspace: %w", err)
		}

		fmt.Printf("%s VM cloudspace '%s' updated successfully\n", color.GreenString("✓"), name)
		return nil
	},
}

// loadVMCSParamsFromFlags loads parameters from command line flags or config file
func loadVMCSParamsFromFlags(cmd *cobra.Command) (*createVMCloudSpaceParams, error) {
	params := &createVMCloudSpaceParams{}

	// Check if config file is provided
	configPath, _ := cmd.Flags().GetString("config")
	if configPath != "" {
		content, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		var fullConfig struct {
			VMCloudSpace struct {
				Name        string              `json:"name" yaml:"name"`
				Org         string              `json:"org" yaml:"org"`
				Region      string              `json:"region" yaml:"region"`
				VMSshKeyRef rxtspot.VMSshKeyRef `json:"vmSshKeyRef" yaml:"vmSshKeyRef"`
				Webhook     string              `json:"webhook,omitempty" yaml:"webhook,omitempty"`
			} `json:"vmCloudSpace" yaml:"vmCloudSpace"`
			VMPools []rxtspot.VMPool `json:"vmPools" yaml:"vmPools"`
		}

		ext := strings.ToLower(filepath.Ext(configPath))
		switch ext {
		case ".yaml", ".yml":
			if err := yaml.Unmarshal(content, &fullConfig); err != nil {
				return nil, fmt.Errorf("failed to unmarshal YAML config: %w", err)
			}
		case ".json":
			if err := json.Unmarshal(content, &fullConfig); err != nil {
				return nil, fmt.Errorf("failed to unmarshal JSON config: %w", err)
			}
		default:
			return nil, fmt.Errorf("unsupported config file format: %s (must be .yaml, .yml, or .json)", ext)
		}

		params.Name = fullConfig.VMCloudSpace.Name
		params.Org = fullConfig.VMCloudSpace.Org
		params.Region = fullConfig.VMCloudSpace.Region
		params.VMSshKeyRef = fullConfig.VMCloudSpace.VMSshKeyRef
		params.Webhook = fullConfig.VMCloudSpace.Webhook
		params.VMPools = fullConfig.VMPools
		return params, nil
	}

	// Load from flags
	params.Name, _ = cmd.Flags().GetString("name")
	params.Org, _ = cmd.Flags().GetString("org")
	params.Region, _ = cmd.Flags().GetString("region")
	params.VMSshKeyRef.Name, _ = cmd.Flags().GetString("vm-ssh-key-name")
	params.VMSshKeyRef.Namespace, _ = cmd.Flags().GetString("vm-ssh-key-namespace")
	params.Webhook, _ = cmd.Flags().GetString("webhook")

	// Parse VM pools
	vmPoolStrs, _ := cmd.Flags().GetStringArray("vm-pool")
	for _, poolStr := range vmPoolStrs {
		poolParams, err := parseNodepoolParams(poolStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse VM pool params: %w", err)
		}

		var pool rxtspot.VMPool
		pool.Name = uuid.NewString()
		if v, ok := poolParams["serverclass"]; ok {
			pool.ServerClass = v
		} else {
			return nil, fmt.Errorf("serverclass is required for VM pool")
		}
		if v, ok := poolParams["bidprice"]; ok {
			validated, err := validateBidPrice(v)
			if err != nil {
				return nil, fmt.Errorf("invalid bid price: %w", err)
			}
			pool.BidPrice = validated
		} else {
			return nil, fmt.Errorf("bidprice is required for VM pool")
		}
		if v, ok := poolParams["desired"]; ok {
			desired, err := validateDesiredCount(v)
			if err != nil {
				return nil, fmt.Errorf("invalid desired count: %w", err)
			}
			pool.Desired = desired
		} else {
			pool.Desired = 1 // defaults to 1 instance
		}
		if v, ok := poolParams["pooltype"]; ok {
			pool.PoolType = v
		}
		if v, ok := poolParams["vmimage"]; ok {
			pool.VMImage = v
		}
		if v, ok := poolParams["vmuserdata"]; ok {
			pool.VMUserData = rxtspot.PrepareUserData(v)
		}

		params.VMPools = append(params.VMPools, pool)
	}

	return params, nil
}
