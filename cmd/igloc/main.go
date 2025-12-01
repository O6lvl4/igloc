package main

import (
	"fmt"
	"os"

	"github.com/O6lvl4/igloc/internal/cli"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "igloc",
		Short: "Scan your local machine and measure how many secrets are hiding there",
		Long: `igloc - Scan your local machine and measure how many secrets are hiding there

igloc helps you discover files that are ignored by .gitignore in your
repositories. This is useful for understanding what secrets and local
configurations exist on your machine that won't be committed to version control.

Use cases:
- Find all .env files hiding in your projects
- Audit what files are excluded from git
- Help set up a new machine by listing required secret files`,
		Version: version,
	}

	rootCmd.AddCommand(cli.NewScanCmd())
	rootCmd.AddCommand(cli.NewSyncCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
