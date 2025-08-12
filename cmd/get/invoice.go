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
var invoiceCmd = &cobra.Command{
	Use:     "invoice <product_id>",
	Short:   "Get stock details for a product",
	Aliases: []string{"invoices", "inv"}, // added alias
	Long:    `Fetches invoice information for the given invoice ID.`,
	Args:    cobra.ExactArgs(1), // Ensures exactly one argument is accepted
	Run: func(cmd *cobra.Command, args []string) {
		invoiceID := args[0]
		if invoiceID == "all" {
			getInvoices()

		}
		fmt.Printf("Fetching invoice details for invoice ID: %s\n", invoiceID)
		// Add your logic here to fetch and display stock info
	},
}

func init() {
	GetCmd.AddCommand(invoiceCmd)
}

func getInvoices() {
	data, err := pkg.Stocks()
	if err != nil {
		fmt.Println("Error fetching stock data:", err)
		return
	}

	var stocks []Stocks

	json.Unmarshal(data, &stocks)
	if err != nil {
		fmt.Println("Error unmarshalling stock data:", err)
		return
	}
	invWr := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(invWr, "INVOICE,\tFOR,\tTYPE,\tPRODUCT ID,\tTOTAL,\tAMOUNT,\tDATE,\tIS PAID")
	for _, stock := range stocks {
		//	ammount := stock.Product.Total * stock.Product.Price
		if stock.Invoice != "" || stock.Type != "NA" {
			//	fmt.Fprintf(invWr, "%s\t%s\t%s\t%s\t%d\t%d\t%s\t%t\n", stock.Invoice, stock.For, stock.Type, stock.Product.UID, stock.Product.Total, stock.Date, stock.IsPaid)
		}
	}
	invWr.Flush()
}
