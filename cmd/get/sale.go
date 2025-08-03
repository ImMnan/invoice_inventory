package get

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/immnan/invoice_invoice/pkg"
	"github.com/spf13/cobra"
)

// stockCmd represents the stock command
var saleCmd = &cobra.Command{
	Use:     "sale",
	Short:   "Get sale details for a product",
	Aliases: []string{"sales", "sl"}, // added alias
	Long:    `Fetches sale information for the given product ID. If no product_id is provided, shows all sales.`,
	Args:    cobra.MaximumNArgs(1), // Allows 0 or 1 argument
	Run: func(cmd *cobra.Command, args []string) {
		var productID string
		if len(args) == 0 {
			productID = "all" // Default to "all" if no argument provided
		} else {
			productID = args[0]
		}
		colorFlag, _ := cmd.Flags().GetString("color")
		printedFlag, _ := cmd.Flags().GetBool("printed")

		printSales(productID, colorFlag, printedFlag)
	},
}

func init() {
	GetCmd.AddCommand(saleCmd)
	saleCmd.Flags().BoolP("help", "h", false, "Help message for toggle")
	saleCmd.Flags().Bool("csv", false, "Output in CSV format")
	saleCmd.Flags().BoolP("printed", "p", false, "Show printed sale values")
	//	saleCmd.Flags().BoolP("rejected", "r", false, "Show rejected sale values")
	saleCmd.Flags().StringP("color", "c", "", "Show sale values for specific color only (e.g., red, green, blue)")
}

type SaleFilter struct {
	ProductID string
	ColorFlag string
	ShowAll   bool
	Printed   bool
}

func printSales(productID, colorFlag string, printedFlag bool) {
	data, err := pkg.Stocks()
	if err != nil {
		fmt.Println("Error fetching stock data:", err)
		return
	}

	var stocks []Stocks
	err = json.Unmarshal(data, &stocks)
	if err != nil {
		fmt.Println("Error unmarshalling stock data:", err)
		return
	}

	tabWriter := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer tabWriter.Flush()

	// Print header once
	fmt.Fprintln(tabWriter, "PRODUCTID\tINVOICE\tTYPE\tCOLOR\tDESIGN\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tTOTAL")

	// Create filter based on input parameters
	filter := StockFilter{
		ProductID: productID,
		ColorFlag: colorFlag,
		ShowAll:   productID == "all" || productID == "",
		Printed:   printedFlag,
	}

	for _, stock := range stocks {
		if !filter.shouldShowSales(stock) {
			continue
		}
		if !filter.shouldShowPrinted(stock) {
			continue
		}
		// Iterate through each color for this stock item
		for colorName, sizeArray := range stock.Product.Color {
			if !filter.shouldShowColor(colorName) {
				continue
			}

			sizes := prepareSizes(sizeArray)
			total := calculateTotal(sizes)
			printStockRow(tabWriter, stock, colorName, sizes, total)
		}
	}
}
