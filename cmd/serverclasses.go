package cmd

import (
	"fmt"

	"github.com/rackspace-spot/spotctl/internal"
	"github.com/rackspace-spot/spotctl/internal/app"
	featserverclasses "github.com/rackspace-spot/spotctl/internal/features/serverclasses"
	"github.com/spf13/cobra"
)

var serverclassesCmd = &cobra.Command{
	Use:     "serverclasses",
	Short:   "Manage serverclasses",
	Long:    `Manage Rackspace Spot serverclasses.`,
	Aliases: []string{"sc", "serverclass"},
}

var serverclassesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List serverclasses",
	Long:  `List all serverclasses.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		region, _ := cmd.Flags().GetString("region")
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{Region: region, RequireRegion: true})
		if err != nil {
			return err
		}
		serverclasses, err := featserverclasses.List(cmd.Context(), appCtx, appCtx.Region)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		return internal.OutputData(serverclasses, outputFormat)
	},
}

var serverclassesGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get serverclass",
	Long:  `Get a specific serverclass.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{})
		if err != nil {
			return err
		}
		serverclass, err := featserverclasses.Get(cmd.Context(), appCtx, name)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		return internal.OutputData(serverclass, outputFormat)
	},
}

func init() {
	rootCmd.AddCommand(serverclassesCmd)
	serverclassesCmd.AddCommand(serverclassesListCmd)
	serverclassesCmd.AddCommand(serverclassesGetCmd)

	serverclassesGetCmd.Flags().String("name", "", "Serverclass name")
	serverclassesGetCmd.MarkFlagRequired("name")

	serverclassesListCmd.Flags().StringP("region", "r", "", "Region name")
	serverclassesListCmd.Flags().StringP("output", "o", "json", "Output format (json, table, yaml)")
}
