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
		approve, _ := cmd.Flags().GetBool("approve")
		csv := false
		csv, _ = cmd.Flags().GetBool("csv")

		if err != nil {
			// Handle error
			return
		}
		if file != "" {
			applyProforma(file, approve, csv)
		} else {
			cmd.Help()
		}
	},
}

func init() {
	ApplyCmd.Flags().StringP("file", "f", "", "File to apply changes from")
	ApplyCmd.Flags().Bool("approve", false, "[!] Approve the changes")
	ApplyCmd.Flags().BoolP("csv", "c", false, "Output in CSV format")
}

// Execute adds all child commands to the root command and sets flags appropriately.

func applyProforma(fileName string, confirm, csv bool) {

	var data *pkg.FileData
	if fileName != "" {
		data = &pkg.FileData{Data: fileName}
	}
	//var productData *pkg.ProductSlice
	stockUpdate, err := data.UpdateStock()
	if err != nil {
		panic(err)
	}

	err := data.ProcessInventoryUpdate(stockUpdate)
	if err != nil {
		panic(err)
	}

}
