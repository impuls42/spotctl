package cmd

import (
	"fmt"

	"github.com/rackspace-spot/spotctl/internal"
	"github.com/rackspace-spot/spotctl/internal/app"
	featpricing "github.com/rackspace-spot/spotctl/internal/features/pricing"
	"github.com/spf13/cobra"
)

var pricingCmd = &cobra.Command{
	Use:   "pricing",
	Short: "Manage pricing",
	Long:  `Manage Rackspace Spot pricing.`,
}

var pricingGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get market price for a server class",
	Long:  `Get market price for a server class.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{})
		if err != nil {
			return err
		}
		serverclass, _ := cmd.Flags().GetString("serverclass")

		pr, err := featpricing.GetForServerClass(cmd.Context(), appCtx, serverclass)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		return internal.OutputData(pr, outputFormat)
	},
}

var pricingGetAllServerClassCmd = &cobra.Command{
	Use:   "get-all",
	Short: "Get market price for all server classes",
	Long:  `Get market price for all server classes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{})
		if err != nil {
			return err
		}

		pr, err := featpricing.GetAll(cmd.Context(), appCtx)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		return internal.OutputData(pr, outputFormat)
	},
}

func init() {
	rootCmd.AddCommand(pricingCmd)
	pricingCmd.AddCommand(pricingGetCmd)
	pricingCmd.AddCommand(pricingGetAllServerClassCmd)
	pricingGetCmd.Flags().String("serverclass", "", "Serverclass name")
}
