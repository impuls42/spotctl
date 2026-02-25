package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/rackspace-spot/spotctl/internal/version"

	"github.com/spf13/cobra"
	"k8s.io/klog"
)

var (
	outputFormat string
	verbosity    int
)

// rootCmd is initialized via NewRootCmd so it can be recreated in tests/other entrypoints.
var rootCmd = NewRootCmd()

// NewRootCmd constructs the root Cobra command.
// It must not call os.Exit; only Execute() should decide process exit.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "spotctl",
		Short:   "Rackspace Spot CLI - Manage your Spot resources",
		Long:    `A command-line interface for managing Rackspace Spot resources. This CLI provides an easy way to manage cloudspaces, node pools, and other Spot resources.`,
		Version: version.GetVersion(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			initLoggingFlags(verbosity)
			klog.V(1).Infof("Verbosity set to %d", verbosity)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Global flags
	cmd.PersistentFlags().IntVarP(&verbosity, "v", "v", 0, "Log verbosity level (0=Errors only)")
	cmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "Output format (json, table, yaml)")

	// Customize the version output format
	cmd.SetVersionTemplate("{{.Name}} version : {{.Version}}\n")

	// Silence usage globally; let Cobra show usage only on flag/arg parsing errors
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// For all runtime errors, just print them cleanly
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		defer klog.Flush() // ensure logs are written before exit
		os.Exit(1)
	}
}

func initLoggingFlags(verbosity int) {
	// Reset the default global FlagSet to avoid "flag redefined" panic
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Initialize klog flags into the new flag set
	klog.InitFlags(nil)

	// Apply verbosity from CLI flag to klog
	_ = flag.Set("v", fmt.Sprintf("%d", verbosity))

	// Always log to stderr (otherwise klog can log to files by default)
	_ = flag.Set("logtostderr", "true")
}
