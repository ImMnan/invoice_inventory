package get

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/immnan/invoice_invoice/pkg"
	"github.com/spf13/cobra"
)

// stockCmd represents the stock command
var invoiceCmd = &cobra.Command{
	Use:     "invoice <invoice_id> [invoice_id ...]",
	Short:   "Get invoice details for one or more invoices",
	Aliases: []string{"invoices", "inv"},
	Long:    `Fetches invoice information for the given invoice ID(s). Accepts one or more IDs separated by spaces, e.g. 'lvs get invoices 123 145 526'`,
	Args:    cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		invoices := []string{}
		format, _ := cmd.Flags().GetString("csv")
		month, _ := cmd.Flags().GetInt("month")
		if len(args) == 0 {
			// Show all invoices if no argument is provided
			getInvoices(format, month)
			return
		}
		for _, id := range args {
			trimmed := strings.TrimSpace(id)
			if trimmed != "" {
				invoices = append(invoices, trimmed)
			}
		}
		if len(invoices) == 0 {
			fmt.Println("No valid invoice IDs provided.")
			return
		}
		// Print invoices for the given IDs
		err := PrintInvoice(format, invoices, month)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	},
}

func init() {
	GetCmd.AddCommand(invoiceCmd)
	invoiceCmd.Flags().String("csv", "table", "Output format (table or csv)")
	invoiceCmd.Flags().IntP("month", "m", 0, "Month to fetch invoices for (default is current month)")
}

func getInvoices(format string, month int) {

	inventoryDB, customerDB, invoiceDB, productDB, err := ConfigData(month)
	if err != nil {
		fmt.Println("Error fetching config data:", err)
		return
	}
	existData := &pkg.JsLocalDB{
		InventoryFile: inventoryDB,
		CustomerFile:  customerDB,
		InvoiceFile:   invoiceDB,
		ProductFile:   productDB,
	}

	data, err := existData.Invoices()
	if err != nil {
		fmt.Println("Error fetching invoice data:", err)
		return
	}

	var invoices []pkg.Invoice

	json.Unmarshal(data, &invoices)

	if format == "table" {
		invWr := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(invWr, "INVOICE\tFOR\tTYPE\tAMOUNT\tDATE\tIS PAID")
		for _, invoice := range invoices {
			if invoice.InvoiceID != "" || invoice.Type != "NA" {
				fmt.Fprintf(invWr, "%s\t%s\t%s\t%d\t%s\t%t\n", invoice.InvoiceID, invoice.Customer.Name, invoice.Type, invoice.Amount, invoice.Date, invoice.IsPaid)
			}
		}
		invWr.Flush()
	}
	if format == "csv" {
		invWr := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(invWr, "INVOICE,\tFOR,\tTYPE,\tAMOUNT,\tDATE,\tIS PAID")
		for _, invoice := range invoices {
			if invoice.InvoiceID != "" || invoice.Type != "NA" {
				fmt.Fprintf(invWr, "%s,\t%s,\t%s,\t%d,\t%s,\t%t\n", invoice.InvoiceID, invoice.Customer.Name, invoice.Type, invoice.Amount, invoice.Date, invoice.IsPaid)
			}
		}
		invWr.Flush()
	}

}

// printInvoice prints the invoice in table or csv format
func PrintInvoice(format string, invoiceID []string, month int) error {

	var invoices []pkg.Invoice

	_, _, invoiceDB, _, err := ConfigData(month)
	if err != nil {
		return err
	}
	invoiceData, err := os.Open(invoiceDB)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(invoiceData)
	if err != nil {
		return err
	}
	defer invoiceData.Close()
	if err := json.Unmarshal(data, &invoices); err != nil {
		return err
	}

	for _, id := range invoiceID {
		found := false
		for _, invoice := range invoices {
			if id == invoice.InvoiceID {
				found = true
				switch format {
				case "table":
					fmt.Println("\n\nFROM: SHIRIKRISHNA TECH\nGST: 1234567890\nCOO: INDIA\nCONTACT: 9725359497\n---")
					fmt.Printf("\nTYPE: %s,\nINVOICE: %s,\nNAME: %s,\nADDRESS: %s,\nGST NUMBER: %s,\nDATE: %s\n\n",
						invoice.Type,
						invoice.InvoiceID,
						invoice.Customer.Name,
						invoice.Customer.Address,
						invoice.Customer.GstNumber,
						invoice.Date)
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
					fmt.Fprintln(w, "PRODUCT Name\tPRICE\tPRINT\tCOLOR\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tQTY\tTOTAL")
					fmt.Fprintln(w, "------------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t--\t----")
					var total int
					var qtyTotal int
					var xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total int
					for _, p := range invoice.Product {
						for color, quantities := range p.Color {
							line := fmt.Sprintf("%s\t%d\t%s\t%s",
								p.Name,
								p.Price,
								p.Print,
								color,
							)
							// Ensure we have 8 sizes: XS, S, M, L, XL, 2X, 3X, 4X
							// If not, pad with zeros
							padded := make([]int, 8)
							copy(padded, quantities)
							xsTotal += padded[0]
							sTotal += padded[1]
							mTotal += padded[2]
							lTotal += padded[3]
							xlTotal += padded[4]
							x2Total += padded[5]
							x3Total += padded[6]
							x4Total += padded[7]
							for _, q := range padded {
								line += fmt.Sprintf("\t%d", q)
							}
							line += fmt.Sprintf("\t%d\t%d", p.Quantity, p.Total)
							fmt.Fprintln(w, line)
							total += p.Total
							qtyTotal += p.Quantity
						}
					}
					fmt.Fprintln(w, "------------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t--\t----")
					fmt.Fprintf(w, "FINAL\t\t\t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total, qtyTotal, total)

					w.Flush()
					finalAmount := total + invoice.TaxAmount
					fmt.Printf("\ncGST:  %s\nsGST:  %s\nTAX:   %d INR\nTAX INVOICE AMOUNT: %d INR\n", "2.5%", "2.5%", invoice.TaxAmount, finalAmount)

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
							fmt.Printf("\nTOTAL, %d\ncGST, %s\nsGST, %s\nTAX, %d INR\nTAX INVOICE AMOUNT, %d INR\n", total, "2.5%", "2.5%", taxAmmount, invoice.TaxAmount)
						}
					}
					w.Flush()
				}
				break
			}
		}
		if !found {
			err := fmt.Errorf("no invoice found with invoice id %s", id)
			return err
		}
	}
	return nil
}
