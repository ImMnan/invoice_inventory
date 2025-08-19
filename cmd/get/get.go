package get

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// getCmd represents the get command
var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Use get command for listing the resources within Blazemeter",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {

}

type Stocks struct {
	UUID     string        `json:"uuid"`
	Type     string        `json:"type"`
	Invoice  string        `json:"invoice"`
	For      string        `json:"for,omitempty"`
	From     string        `json:"from,omitempty"`
	Date     string        `json:"date,omitempty"`
	IsPaid   bool          `json:"isPaid"`
	Rejected bool          `json:"rejected"`
	Product  ProductStruct `json:"product"`
}

type ProductStruct struct {
	ProductID   string           `json:"product_id"`
	Name        string           `json:"name"`
	Print       string           `json:"print"`
	Gen         string           `json:"gen"`
	GST         int              `json:"gst"`
	Color       map[string][]int `json:"color"`
	Total       int              `json:"total"`
	Price       int              `json:"price"`
	Description string           `json:"description"`
	Quantity    int              `json:"quantity"`
}

// shouldShowStock determines if a stock should be displayed based on the filter
func (sf StockFilter) shouldShowStock(stock Stocks) bool {
	if stock.Type != "in_stock" {
		return false
	}

	// If showing all stocks
	if sf.ShowAll {
		return true
	}

	// If specific product ID is requested
	if sf.ProductID != "" && sf.ProductID != "all" {
		return stock.Product.ProductID == sf.ProductID
	}

	return false
}

func (sf StockFilter) shouldShowSales(stock Stocks) bool {
	if stock.Type != "sale" {
		return false
	}

	// If showing all stocks
	if sf.ShowAll {
		return true
	}

	// If specific product ID is requested
	if sf.ProductID != "" && sf.ProductID != "all" {
		return stock.Product.ProductID == sf.ProductID
	}

	return false
}

func (sf StockFilter) shouldShowPurchases(stock Stocks) bool {
	if stock.Type != "purchase" {
		return false
	}

	// If showing all stocks
	if sf.ShowAll {
		return true
	}

	// If specific product ID is requested
	if sf.ProductID != "" && sf.ProductID != "all" {
		return stock.Product.ProductID == sf.ProductID
	}

	return false
}

// shouldShowColor determines if a specific color should be displayed
func (sf StockFilter) shouldShowColor(colorName string) bool {
	if sf.ColorFlag == "" {
		return true
	}
	return colorName == sf.ColorFlag
}

func (sf StockFilter) shouldShowPrinted(stock Stocks) bool {
	// If printed flag is not set, show all stocks regardless of printed status
	if !sf.Printed {
		return true
	}
	// If printed flag is set, only show stocks that have been printed
	return stock.Product.Print != ""
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
func printStockRow(tabWriter *tabwriter.Writer, stock Stocks, colorName string, sizes []int, total int) {
	fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
		stock.Product.ProductID,
		stock.Invoice,
		stock.Type,
		colorName,
		stock.Product.Print,
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

func ConfigData(month int) (inventoryDB, customerDB, invoiceDB, productDB string, logErr error) {

	vp := viper.New()
	vp.SetConfigName("lvsConfig")
	vp.SetConfigType("yaml")
	//	vp.AddConfigPath(".")
	vp.AddConfigPath(".")
	err := vp.ReadInConfig()
	if err != nil {
		return "", "", "", "", err
	}
	var inventoryData, customerData, invoiceData, productData string
	monthSuffixes := []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	suffix := monthSuffixes[month]
	prefix := "data"
	if suffix != "" {
		prefix += suffix
	}
	inventoryData = vp.GetString(prefix + ".inventoryData")
	customerData = vp.GetString(prefix + ".customersData")
	invoiceData = vp.GetString(prefix + ".invoicesData")
	productData = vp.GetString(prefix + ".productsData")

	return inventoryData, customerData, invoiceData, productData, nil
}
