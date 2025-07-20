package get

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stockCmd represents the stock command
var invoiceCmd = &cobra.Command{
	Use:     "invoice <product_id>",
	Short:   "Get stock details for a product",
	Aliases: []string{"invoices", "inv"}, // added alias
	Long:    `Fetches invoice information for the given invoice ID.`,
	Args:    cobra.ExactArgs(1), // Ensures exactly one argument is accepted
	Run: func(cmd *cobra.Command, args []string) {
		invoiceID := args[0]
		if invoiceID == "all" {

			fmt.Println("Function to print all invoice values for that threshold time duration")

		}
		fmt.Printf("Fetching invoice details for invoice ID: %s\n", invoiceID)
		// Add your logic here to fetch and display stock info
	},
}

func init() {
	GetCmd.AddCommand(invoiceCmd)
}
