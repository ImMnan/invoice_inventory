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

// printInvoice prints the invoice in table or csv format
func PrintInvoice(format string, invoiceID []string) error {

	var invoice pkg.Invoice

	invoiceFile, err := os.ReadFile("Data/invoices.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(invoiceFile, &invoice); err != nil {
		return err
	}
	for _, i := range invoiceID {
		if i == invoice.InvoiceID {

			switch format {
			case "table":
				fmt.Println("\nFROM: SHIRIKRISHNA TECH\nGST: 1234567890\nCOO: INDIA\nCONTACT: 9725359497\n---")
				fmt.Printf("TYPE: %s,\nINVOICE: %s,\nNAME: %s,\nADDRESS: %s,\nGST NUMBER: %s,\nDATE: %s\n\n",
					invoice.Type,
					invoice.InvoiceID,
					invoice.Customer.Name,
					invoice.Customer.Address,
					invoice.Customer.GstNumber,
					invoice.Date)
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "PRODUCT Name\tPRICE\tPRINT\tCOLOR\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tQTY\tTOTAL")
				fmt.Fprintln(w, "------------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t---\t----")
				for _, p := range invoice.Product {
					for color, quantities := range p.Color {
						qty := 0
						for _, q := range quantities {
							qty += q
						}
						line := fmt.Sprintf("%s\t%d\t%s\t%s",
							p.Name,
							p.Price,
							p.Print,
							color,
						)
						for _, q := range quantities {
							line += fmt.Sprintf("\t%d", q)
						}
						total := p.Price * qty
						line += fmt.Sprintf("\t%d\t%d", qty, total)
						fmt.Fprintln(w, line)
						taxAmmount := total * 5 / 100
						fmt.Printf("\nTOTAL: %d\ncGST: %s\niGST: %s\nTAX: %d\nTAX INVOICE AMOUNT: %d\n---", total, "2.5%", "2.5%", taxAmmount, invoice.TaxAmount)

					}
				}
				w.Flush()
			case "csv":
				fmt.Println("\nFROM, SHIRIKRISHNA TECH,\nGST, 1234567890,\nCOO, INDIA\nCONTACT, 9725359497")
				fmt.Printf("\nTYPE,%s,\nINVOICE,%s,\nNAME,%s,\nADDRESS,%s,\nGST NUMBER,%s,\nDATE,%s\n\n",
					invoice.Type,
					invoice.InvoiceID,
					invoice.Customer.Name,
					invoice.Customer.Address,
					invoice.Customer.GstNumber,
					invoice.Date)
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "PRODUCT ID,\tPRODUCT Name,\tPRICE,\tPRINT,\tCOLOR,\tXS,\tS,\tM,\tL,\tXL,\t2X,\t3X,\t4X,\tQTY,\tTOTAL")
				for _, p := range invoice.Product {
					for color, quantities := range p.Color {
						qty := 0
						for _, q := range quantities {
							qty += q
						}
						line := fmt.Sprintf("%s\t%d\t%s\t%s",
							p.Name,
							p.Price,
							p.Print,
							color,
						)
						for _, q := range quantities {
							line += fmt.Sprintf("\t%d", q)
						}

						total := p.Price * qty
						line += fmt.Sprintf("\t%d\t%d", qty, total)
						fmt.Fprintln(w, line)
						taxAmmount := total * 5 / 100
						fmt.Printf("\nTOTAL, %d\ncGST, %s\niGST, %s\nTAX, %d\nTAX INVOICE AMOUNT, %d\n", total, "2.5%", "2.5%", taxAmmount, invoice.TaxAmount)

					}
				}
				w.Flush()
			}
		} else {
			err := fmt.Errorf("no invoice found with invoice id %s", invoiceID)
			return err
		}
	}
	return nil
}
