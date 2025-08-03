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
var stockCmd = &cobra.Command{
	Use:     "stock [product_id]",
	Short:   "Get stock details for a product",
	Aliases: []string{"stocks", "stk"}, // added alias
	Long:    `Fetches stock information for the given product ID. If no product_id is provided, shows all stocks.`,
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

		printStocks(productID, colorFlag, printedFlag)
	},
}

func init() {
	GetCmd.AddCommand(stockCmd)
	stockCmd.Flags().BoolP("help", "h", false, "Help message for toggle")
	stockCmd.Flags().Bool("csv", false, "Output in CSV format")
	stockCmd.Flags().BoolP("printed", "p", false, "Show printed stock values")
	//	stockCmd.Flags().BoolP("rejected", "r", false, "Show rejected stock values")
	stockCmd.Flags().StringP("color", "c", "", "Show stock values for specific color only (e.g., red, green, blue)")
}

// StockFilter defines the filtering criteria for stocks
type StockFilter struct {
	ProductID string
	ColorFlag string
	ShowAll   bool
	Printed   bool
}

// shouldShowStock determines if a stock should be displayed based on the filter
func (sf StockFilter) shouldShowStock(stock pkg.TshirtStruct) bool {
	if stock.Type != "in_stock" {
		return false
	}

	// If showing all stocks
	if sf.ShowAll {
		return true
	}

	// If specific product ID is requested
	if sf.ProductID != "" && sf.ProductID != "all" {
		return stock.ProductID == sf.ProductID
	}

	return false
}

// shouldShowColor determines if a specific color should be displayed
func (sf StockFilter) shouldShowColor(stock pkg.TshirtStruct, colorName string) bool {
	if sf.ColorFlag == "" {
		return true
	}
	return colorName == sf.ColorFlag
}

func (sf StockFilter) shouldShowPrinted(stock pkg.TshirtStruct) bool {
	// If printed flag is not set, show all stocks regardless of printed status
	if !sf.Printed {
		return true
	}
	// If printed flag is set, only show stocks that have been printed
	return stock.Print != ""
}

// prepareSizes ensures we have exactly 8 size values, padding with 0 if necessary
func prepareSizes(sizeArray []int) []int {
	sizes := make([]int, 8)
	for i := 0; i < len(sizeArray) && i < 8; i++ {
		sizes[i] = sizeArray[i]
	}
	return sizes
}

// calculateTotal calculates the total quantity across all sizes
func calculateTotal(sizes []int) int {
	total := 0
	for _, qty := range sizes {
		total += qty
	}
	return total
}

// printStockRow prints a single row of stock data to the tabwriter
func printStockRow(tabWriter *tabwriter.Writer, stock pkg.TshirtStruct, colorName string, sizes []int, total int) {
	fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
		stock.ProductID,
		stock.Type,
		colorName,
		stock.Print,
		sizes[0], // XS
		sizes[1], // S
		sizes[2], // M
		sizes[3], // L
		sizes[4], // XL
		sizes[5], // 2XL
		sizes[6], // 3XL
		sizes[7], // 4XL
		total)
}

func printStocks(productID, colorFlag string, printedFlag bool) {
	data, err := pkg.Stocks()
	if err != nil {
		fmt.Println("Error fetching stock data:", err)
		return
	}

	var stocks []pkg.TshirtStruct
	err = json.Unmarshal(data, &stocks)
	if err != nil {
		fmt.Println("Error unmarshalling stock data:", err)
		return
	}

	tabWriter := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer tabWriter.Flush()

	// Print header once
	fmt.Fprintln(tabWriter, "PRODUCTID\tTYPE\tCOLOR\tDESIGN\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tTOTAL")

	// Create filter based on input parameters
	filter := StockFilter{
		ProductID: productID,
		ColorFlag: colorFlag,
		ShowAll:   productID == "all" || productID == "",
		Printed:   printedFlag,
	}

	for _, stock := range stocks {
		if !filter.shouldShowStock(stock) {
			continue
		}
		if !filter.shouldShowPrinted(stock) {
			continue
		}
		// Iterate through each color for this stock item
		for colorName, sizeArray := range stock.Color {
			if !filter.shouldShowColor(stock, colorName) {
				continue
			}

			sizes := prepareSizes(sizeArray)
			total := calculateTotal(sizes)
			printStockRow(tabWriter, stock, colorName, sizes, total)
		}
	}
}
