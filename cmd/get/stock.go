package get

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stockCmd represents the stock command
var stockCmd = &cobra.Command{
	Use:     "stock <product_id>",
	Short:   "Get stock details for a product",
	Aliases: []string{"stocks", "stk"}, // 👈 Add "stocks" as an alias
	Long:    `Fetches stock information for the given product ID.`,
	Args:    cobra.ExactArgs(1), // Ensures exactly one argument is passed
	Run: func(cmd *cobra.Command, args []string) {
		productID := args[0]
		fmt.Printf("Fetching stock details for product ID: %s\n", productID)
		// Add your logic here to fetch and display stock info
	},
}

func init() {
	stockCmd.Flags().BoolP("help", "h", false, "Help message for toggle")
	GetCmd.AddCommand(stockCmd)

}
