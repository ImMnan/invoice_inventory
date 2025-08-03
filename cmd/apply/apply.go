package apply

import (
	"github.com/immnan/invoice_invoice/pkg"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var ApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Use get command for listing the resources within Blazemeter",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		file, err := cmd.Flags().GetString("file")
		if err != nil {
			// Handle error
			return
		}
		if file != "" {
			pkg.ApplyProforma(file)

		} else {
			cmd.Help()
		}
	},
}

func init() {
	ApplyCmd.Flags().StringP("file", "f", "", "File to apply changes from")
}

// Execute adds all child commands to the root command and sets flags appropriately.
