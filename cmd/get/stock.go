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
		month, _ := cmd.Flags().GetInt("month")
		colorFlag, _ := cmd.Flags().GetString("color")
		printedFlag, _ := cmd.Flags().GetBool("printed")
		var productID string

		if len(args) == 0 {
			productID = "all" // Default to "all" if no argument provided
		} else {
			productID = args[0]
		}
		printStocks(productID, colorFlag, printedFlag, month)
	},
}

func init() {
	GetCmd.AddCommand(stockCmd)
	stockCmd.Flags().BoolP("help", "h", false, "Help message for toggle")
	stockCmd.Flags().Bool("csv", false, "Output in CSV format")
	stockCmd.Flags().BoolP("printed", "p", false, "Show printed stock values")
	//	stockCmd.Flags().BoolP("rejected", "r", false, "Show rejected stock values")
	stockCmd.Flags().StringP("color", "c", "", "Show stock values for specific color only (e.g., red, green, blue)")
	stockCmd.Flags().IntP("month", "m", 0, "Month to fetch stocks for (default is current month)")
}

// StockFilter defines the filtering criteria for stocks
type StockFilter struct {
	ProductID string
	ColorFlag string
	ShowAll   bool
	Printed   bool
}

func printStocks(productID, colorFlag string, printedFlag bool, month int) {

	inventoryDB, customerDB, invoiceDB, productDB, err := ConfigData(month) // Assuming 0 for current month
	if err != nil {
		fmt.Println("Error fetching config data:", err)
		return
	}

	existData := &pkg.JsLocalDB{InventoryFile: inventoryDB, CustomerFile: customerDB, InvoiceFile: invoiceDB, ProductFile: productDB}
	data, err := existData.Stocks()
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
		if !filter.shouldShowStock(stock) {
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
