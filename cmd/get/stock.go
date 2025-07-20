package get

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stockCmd represents the stock command
var stockCmd = &cobra.Command{
	Use:     "stock <product_id>",
	Short:   "Get stock details for a product",
	Aliases: []string{"stocks", "stk"}, // added alias
	Long:    `Fetches stock information for the given product ID.`,
	Args:    cobra.ExactArgs(1), // Ensures exactly one argument is accepted
	Run: func(cmd *cobra.Command, args []string) {
		productID := args[0]
		if productID == "all" {

			fmt.Println("Function that would print the stock values")

		}
		fmt.Printf("Fetching stock details for product ID: %s\n", productID)

	},
}

func init() {
	GetCmd.AddCommand(stockCmd)
	stockCmd.Flags().BoolP("help", "h", false, "Help message for toggle")
	stockCmd.Flags().Bool("csv", false, "Output in CSV format")
	stockCmd.Flags().Bool("printed", false, "Show printed stock values")
	stockCmd.Flags().BoolP("rejected", "r", false, "Show rejected stock values")
	stockCmd.Flags().StringP("color", "c", "", "Show stock values for specific color only (e.g., red, green, blue)")
	stockCmd.Flags().StringP("size", "s", "", "Show stock values for specific size only (e.g., S, M, L, XL, 2XL)")
}
