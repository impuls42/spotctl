package cmd

import (
	"fmt"

	"github.com/rackspace-spot/spotctl/internal"
	"github.com/rackspace-spot/spotctl/internal/app"
	featorgs "github.com/rackspace-spot/spotctl/internal/features/organizations"
	"github.com/spf13/cobra"
)

// organizationsCmd represents the organizations command
var organizationsCmd = &cobra.Command{
	Use:     "organizations",
	Short:   "Manage organizations",
	Long:    `Manage Rackspace Spot organizations (namespaces).`,
	Aliases: []string{"org", "organization"},
}

// organizationsListCmd represents the organizations list command
var organizationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List organizations",
	Long:  `List all organizations accessible by the authenticated user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{})
		if err != nil {
			return err
		}
		orgs, err := featorgs.List(cmd.Context(), appCtx)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		return internal.OutputData(orgs, outputFormat)
	},
}

// organizationsGetCmd represents the organizations get command
var organizationsGetCmd = &cobra.Command{
	Use:   "get <org>",
	Short: "Get organization details",
	Long:  `Get details for a specific organization by org.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{})
		if err != nil {
			return err
		}
		orgName, _ := cmd.Flags().GetString("name")
		if orgName == "" {
			return fmt.Errorf("organization not specified")
		}
		org, err := featorgs.GetByName(cmd.Context(), appCtx, orgName)
		if err != nil {
			return err
		}
		return internal.OutputData(org, outputFormat)
	},
}

func init() {
	rootCmd.AddCommand(organizationsCmd)
	organizationsCmd.AddCommand(organizationsListCmd)
	organizationsCmd.AddCommand(organizationsGetCmd)
	organizationsGetCmd.Flags().String("name", "", "Organization name (required)")

	organizationsGetCmd.MarkFlagRequired("name")
}
