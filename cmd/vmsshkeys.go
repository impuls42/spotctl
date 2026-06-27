package cmd

import (
	"fmt"

	"github.com/fatih/color"
	rxtspot "github.com/rackspace-spot/spot-go-sdk/api/v1"
	"github.com/rackspace-spot/spotctl/internal"
	"github.com/rackspace-spot/spotctl/internal/app"
	featvmsshkeys "github.com/rackspace-spot/spotctl/internal/features/vmsshkeys"
	"github.com/spf13/cobra"
)

// vmSSHKeysCmd represents the vmsshkeys command
var vmSSHKeysCmd = &cobra.Command{
	Use:     "vmsshkeys",
	Short:   "Manage VM SSH keys",
	Long:    `Manage Rackspace Spot VM SSH keys.`,
	Aliases: []string{"vmsshkey", "vmsk"},
}

func init() {
	rootCmd.AddCommand(vmSSHKeysCmd)
	vmSSHKeysCmd.AddCommand(vmSSHKeyListCmd)
	vmSSHKeysCmd.AddCommand(vmSSHKeyCreateCmd)
	vmSSHKeysCmd.AddCommand(vmSSHKeyGetCmd)
	vmSSHKeysCmd.AddCommand(vmSSHKeyDeleteCmd)

	// Flags for vmsshkeys list
	vmSSHKeyListCmd.Flags().String("org", "", "Organization name")

	// Flags for vmsshkeys create
	vmSSHKeyCreateCmd.Flags().String("name", "", "SSH key name (required)")
	vmSSHKeyCreateCmd.Flags().String("org", "", "Organization name")
	vmSSHKeyCreateCmd.Flags().String("public-key", "", "SSH public key (required)")
	vmSSHKeyCreateCmd.Flags().String("description", "", "Description of the SSH key")
	vmSSHKeyCreateCmd.MarkFlagRequired("name")
	vmSSHKeyCreateCmd.MarkFlagRequired("public-key")

	// Flags for vmsshkeys get
	vmSSHKeyGetCmd.Flags().String("name", "", "SSH key name (required)")
	vmSSHKeyGetCmd.Flags().String("org", "", "Organization name")
	vmSSHKeyGetCmd.MarkFlagRequired("name")

	// Flags for vmsshkeys delete
	vmSSHKeyDeleteCmd.Flags().String("name", "", "SSH key name (required)")
	vmSSHKeyDeleteCmd.Flags().String("org", "", "Organization name")
	vmSSHKeyDeleteCmd.MarkFlagRequired("name")
	vmSSHKeyDeleteCmd.Flags().BoolP("yes", "y", false, "Automatic yes to prompts")
}

var vmSSHKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List VM SSH keys",
	Long:  `List all VM SSH keys in an organization.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		org, _ := cmd.Flags().GetString("org")
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{Org: org, RequireOrg: true})
		if err != nil {
			return err
		}

		keys, err := featvmsshkeys.List(cmd.Context(), appCtx, appCtx.Org)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		return internal.OutputData(keys, outputFormat)
	},
}

var vmSSHKeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a VM SSH key",
	Long:  `Create a new VM SSH key.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		org, _ := cmd.Flags().GetString("org")
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{Org: org, RequireOrg: true})
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		publicKey, _ := cmd.Flags().GetString("public-key")
		description, _ := cmd.Flags().GetString("description")

		if err := featvmsshkeys.Create(cmd.Context(), appCtx, featvmsshkeys.CreateParams{
			Org:         appCtx.Org,
			Name:        name,
			PublicKey:   publicKey,
			Description: description,
		}); err != nil {
			return fmt.Errorf("failed to create VM SSH key: %w", err)
		}

		fmt.Printf("\n%s Successfully created VM SSH key %s\n",
			color.GreenString("✓"),
			color.CyanString(name),
		)

		return nil
	},
}

var vmSSHKeyGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get VM SSH key details",
	Long:  `Get details about a specific VM SSH key.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		org, _ := cmd.Flags().GetString("org")
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{Org: org, RequireOrg: true})
		if err != nil {
			return err
		}

		key, err := featvmsshkeys.Get(cmd.Context(), appCtx, appCtx.Org, name)
		if err != nil {
			if rxtspot.IsNotFound(err) {
				return fmt.Errorf("VM SSH key '%s' not found", name)
			}
			return fmt.Errorf("failed to get VM SSH key: %w", err)
		}

		return internal.OutputData(key, outputFormat)
	},
}

var vmSSHKeyDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a VM SSH key",
	Long:  `Delete a VM SSH key.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		org, _ := cmd.Flags().GetString("org")
		appCtx, err := app.Load(cmd.Context(), app.LoadOptions{Org: org, RequireOrg: true})
		if err != nil {
			return err
		}

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			prompt := color.New(color.FgYellow).PrintfFunc()
			prompt("Are you sure you want to delete VM SSH key '%s'? (y/N): ", name)

			var response string
			_, err := fmt.Scanln(&response)
			if err != nil || (response != "y" && response != "Y") {
				fmt.Println("Aborted.")
				return nil
			}
		}

		if err := featvmsshkeys.Delete(cmd.Context(), appCtx, appCtx.Org, name); err != nil {
			if rxtspot.IsNotFound(err) {
				return fmt.Errorf("VM SSH key '%s' not found", name)
			}
			return fmt.Errorf("failed to delete VM SSH key: %w", err)
		}

		fmt.Printf("VM SSH key '%s' deleted successfully\n", name)
		return nil
	},
}
