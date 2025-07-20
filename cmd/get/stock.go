package get

import (
	"encoding/csv"
	"fmt"
	"os"

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
			getStock()
		}
		fmt.Printf("Fetching stock details for product ID: %s\n", productID)
		// Add your logic here to fetch and display stock info
	},
}

func init() {
	stockCmd.Flags().BoolP("help", "h", false, "Help message for toggle")
	GetCmd.AddCommand(stockCmd)

}

func getStock() {
	// Open the CSV file
	file, err := os.Open("Data/inventory.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	// Read the CSV data
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields
	data, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	// Print the CSV data
	for _, row := range data {
		for _, col := range row {
			fmt.Printf("%s,", col)
		}
		fmt.Println()
	}
}
