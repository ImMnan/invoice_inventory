package get

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
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
		formatCsv, _ := cmd.Flags().GetBool("csv")
		month, _ := cmd.Flags().GetInt("month")
		unPaid, _ := cmd.Flags().GetBool("up")
		customer, _ := cmd.Flags().GetString("customer")

		if len(args) == 0 {
			// Show all invoices if no argument is provided
			getInvoices(formatCsv, unPaid, month, customer)
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
		err := PrintInvoice(formatCsv, invoices, month)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	},
}

func init() {
	GetCmd.AddCommand(invoiceCmd)
	invoiceCmd.Flags().Bool("csv", false, "Output format (table or csv)")
	invoiceCmd.Flags().IntP("month", "m", 0, "Month to fetch invoices for (default is current month)")
	invoiceCmd.Flags().Bool("up", false, "Show all unpaid invoices")
	invoiceCmd.Flags().String("customer", "all", "Filter invoices by customer ID or name")
}

func getInvoices(formatCsv, unPaid bool, month int, customer string) {

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

	// Create filter based on input parameters
	filter := InvoiceFilter{
		UnPaid:     unPaid,
		CustomerID: customer,
		ShowAll:    true,
	}

	if !formatCsv {
		invWr := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(invWr, "INVOICE\tFOR\tTYPE\tQTY\tAMOUNT\tDATE\tIS PAID")

		var grandTotalQty int
		var grandTotalAmount int

		for _, invoice := range invoices {
			if !filter.shouldShowInvoice(invoice) {
				continue
			}
			// Calculate total quantity for this invoice
			var totalQty int
			for _, product := range invoice.Product {
				totalQty += product.Quantity
			}
			netAmount := invoice.Amount - invoice.PartialPayment
			fmt.Fprintf(invWr, "%s\t%s\t%s\t%d\t%d\t%s\t%t\n", invoice.InvoiceID, invoice.Customer.Name, invoice.Type, totalQty, netAmount, invoice.Date, invoice.IsPaid)

			// Add to grand totals
			grandTotalQty += totalQty
			grandTotalAmount += netAmount
		}

		// Print totals footer
		fmt.Fprintln(invWr, "----------\t-----\t-----\t-----\t------\t----------\t-------")
		fmt.Fprintf(invWr, "TOTAL\t\t\t%d\t%d\t\t\n", grandTotalQty, grandTotalAmount)
		invWr.Flush()
	}
	if formatCsv {
		invWr := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(invWr, "INVOICE,\tFOR,\tTYPE,\tQTY,\tAMOUNT,\tDATE,\tIS PAID")

		var grandTotalQty int
		var grandTotalAmount int

		for _, invoice := range invoices {
			if !filter.shouldShowInvoice(invoice) {
				continue
			}
			// Calculate total quantity for this invoice
			var totalQty int
			for _, product := range invoice.Product {
				totalQty += product.Quantity
			}
			netAmount := invoice.Amount - invoice.PartialPayment
			fmt.Fprintf(invWr, "%s,\t%s,\t%s,\t%d,\t%d,\t%s,\t%t\n", invoice.InvoiceID, invoice.Customer.Name, invoice.Type, totalQty, netAmount, invoice.Date, invoice.IsPaid)

			// Add to grand totals
			grandTotalQty += totalQty
			grandTotalAmount += netAmount
		}

		// Print totals footer
		fmt.Fprintln(invWr, "----------,\t-----,\t-----,\t-----,\t------,\t----------,\t-------")
		fmt.Fprintf(invWr, "TOTAL,\t,\t,\t%d,\t%d,\t,\t\n", grandTotalQty, grandTotalAmount)
		invWr.Flush()
	}

}

// printInvoice prints the invoice in table or csv format
func PrintInvoice(formatCsv bool, invoiceID []string, month int) error {

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

				// Group products by print name
				type productRow struct {
					Name       string
					Price      int
					Print      string
					Color      string
					Quantities []int
					Quantity   int
					Total      int
				}
				printMap := make(map[string][]productRow)
				for _, p := range invoice.Product {
					for color, quantities := range p.Color {
						row := productRow{
							Name:       p.Name,
							Price:      p.Price,
							Print:      p.Print,
							Color:      color,
							Quantities: make([]int, 8),
							Quantity:   p.Quantity,
							Total:      p.Total,
						}
						copy(row.Quantities, quantities)
						printMap[p.Print] = append(printMap[p.Print], row)
					}
				}

				// To ensure order, collect print names
				var printNames []string
				for printName := range printMap {
					printNames = append(printNames, printName)
				}
				// Sort print names for consistent output
				if len(printNames) > 1 {
					sort.Strings(printNames)
				}

				switch formatCsv {
				case false:
					fmt.Println("\n\nFROM: SHIRIKRISHNA TECH\nGST: 24ERKPP7790D1ZC\nCOO: INDIA\nCONTACT: 9924936211\n---")
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
					for _, printName := range printNames {
						rows := printMap[printName]
						for _, row := range rows {
							line := fmt.Sprintf("%s\t%d\t%s\t%s",
								row.Name,
								row.Price,
								row.Print,
								row.Color,
							)
							padded := row.Quantities
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
							line += fmt.Sprintf("\t%d\t%d", row.Quantity, row.Total)
							fmt.Fprintln(w, line)
							total += row.Total
							qtyTotal += row.Quantity
						}
					}
					fmt.Fprintln(w, "------------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t--\t----")
					fmt.Fprintf(w, "FINAL\t\t\t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total, qtyTotal, total)

					w.Flush()
					finalAmount := total + invoice.TaxAmount
					fmt.Printf("\ncGST:  %s\nsGST:  %s\nTAX:   %d INR\nTAX INVOICE AMOUNT: %d INR\n", "2.5%", "2.5%", invoice.TaxAmount, finalAmount)

					fmt.Println("\n\n[!] This is a computer generated invoice, please report any discrepancies to sales team.")
					fmt.Println("[*] No physical signature is required.")

					fmt.Println("\n\n\n'lvs' Copyright (C) 2025  SHRIKRISHNA TECH")

				case true:
					fmt.Println("\nFROM, SHIRIKRISHNA TECH,\nGST, 1234567890,\nCOO, INDIA\nCONTACT, 9725359497")
					fmt.Printf("\nTYPE,%s,\nINVOICE,%s,\nNAME,%s,\nADDRESS,%s,\nGST NUMBER,%s,\nDATE,%s\n\n",
						invoice.Type,
						invoice.InvoiceID,
						invoice.Customer.Name,
						invoice.Customer.Address,
						invoice.Customer.GstNumber,
						invoice.Date)
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
					fmt.Fprintln(w, "PRODUCT Name,\tPRICE,\tPRINT,\tCOLOR,\tXS,\tS,\tM,\tL,\tXL,\t2X,\t3X,\t4X,\tQTY,\tTOTAL")
					var total int
					var qtyTotal int
					var xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total int
					for _, printName := range printNames {
						rows := printMap[printName]
						for _, row := range rows {
							line := fmt.Sprintf("%s,\t%d,\t%s,\t%s,",
								row.Name,
								row.Price,
								row.Print,
								row.Color,
							)
							padded := row.Quantities
							xsTotal += padded[0]
							sTotal += padded[1]
							mTotal += padded[2]
							lTotal += padded[3]
							xlTotal += padded[4]
							x2Total += padded[5]
							x3Total += padded[6]
							x4Total += padded[7]
							for _, q := range padded {
								line += fmt.Sprintf("\t%d,", q)
							}
							line += fmt.Sprintf("\t%d,\t%d,", row.Quantity, row.Total)
							fmt.Fprintln(w, line)
							total += row.Total
							qtyTotal += row.Quantity
						}
					}
					fmt.Fprintf(w, "FINAL,\t,\t,\t,\t%d,\t%d,\t%d,\t%d,\t%d,\t%d,\t%d,\t%d,\t%d,\t%d\n", xsTotal, sTotal, mTotal, lTotal, xlTotal, x2Total, x3Total, x4Total, qtyTotal, total)

					w.Flush()
					finalAmount := total + invoice.TaxAmount
					fmt.Printf("\ncGST,  %s,\nsGST,  %s,\nTAX,   %d, INR,\nTAX INVOICE AMOUNT, %d, INR\n", "2.5%", "2.5%", invoice.TaxAmount, finalAmount)

					fmt.Println("\n\n[!] This is a computer generated invoice please report any discrepancies to sales team.")
					fmt.Println("[*] No physical signature is required.")

					fmt.Println("\n\n\n'lvs' Copyright (C) 2025  SHRIKRISHNA TECH")

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

// shouldShowInvoice determines if an invoice should be displayed based on the filter
func (inf InvoiceFilter) shouldShowInvoice(invoice pkg.Invoice) bool {
	// Filter by payment status if unpaid flag is set
	if inf.UnPaid && invoice.IsPaid {
		return false
	}

	// Filter by customer ID if specified
	if inf.CustomerID != "" && inf.CustomerID != "all" {
		// Check both customer ID and customer name (case-insensitive)
		customerIDMatch := strings.EqualFold(invoice.Customer.CustomerId, inf.CustomerID)
		customerNameMatch := strings.Contains(strings.ToLower(invoice.Customer.Name), strings.ToLower(inf.CustomerID))
		if !customerIDMatch && !customerNameMatch {
			return false
		}
	}

	// Basic validation - exclude empty or invalid invoices
	if invoice.InvoiceID == "" && invoice.Type == "NA" {
		return false
	}

	return true
}
