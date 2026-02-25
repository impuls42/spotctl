package cmd

import (
	"fmt"

	"github.com/rackspace-spot/spotctl/internal"
	"github.com/rackspace-spot/spotctl/internal/app"
	featregions "github.com/rackspace-spot/spotctl/internal/features/regions"
	"github.com/spf13/cobra"
)

var regionsCmd = &cobra.Command{
	Use:   "regions",
	Short: "Manage regions",
	Long:  `Manage Rackspace Spot regions.`,
}

var regionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List regions",
	Long:  `List all regions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{})
		if err != nil {
			return err
		}
		regions, err := featregions.List(cmd.Context(), appCtx)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		return internal.OutputData(regions, outputFormat)
	},
}

var regionsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get region",
	Long:  `Get a specific region.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("name is required")
		}
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{})
		if err != nil {
			return err
		}
		region, err := featregions.Get(cmd.Context(), appCtx, name)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		return internal.OutputData(region, outputFormat)
	},
}

func init() {
	rootCmd.AddCommand(regionsCmd)
	regionsCmd.AddCommand(regionsListCmd)
	regionsCmd.AddCommand(regionsGetCmd)

	regionsGetCmd.Flags().String("name", "", "Region name")
	regionsListCmd.Flags().StringP("output", "o", "json", "Output format (json, table, yaml)")
}
