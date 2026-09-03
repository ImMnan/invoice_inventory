package update

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/immnan/invoice_invoice/cmd/get"
	"github.com/immnan/invoice_invoice/pkg"
	"github.com/spf13/cobra"
)

// stockCmd represents the stock command
var invoiceCmd = &cobra.Command{
	Use:     "invoice <invoice_id> [invoice_id ...]",
	Short:   "Update invoice details for one or more invoices",
	Aliases: []string{"invoices", "inv"},
	Long:    `Updates invoice information for the given invoice ID(s). Accepts one or more IDs separated by spaces, e.g. 'lvs update invoices 123 145 526'`,
	Args:    cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		month, _ := cmd.Flags().GetInt("month")
		pay, _ := cmd.Flags().GetInt("pay")
		paid, _ := cmd.Flags().GetBool("paid")
		updateInvoiceById(args, month, pay, paid)

	},
}

func init() {
	UpdateCmd.AddCommand(invoiceCmd)
	invoiceCmd.Flags().IntP("month", "m", 0, "Month to fetch invoices for (default is current month)")
	invoiceCmd.Flags().IntP("pay", "p", 0, "Update payment received for the invoice")
	invoiceCmd.Flags().Bool("paid", false, "Confirm all payment received")
}

func updateInvoiceById(invoices []string, month, pay int, paid bool) {
	if len(invoices) == 0 {
		fmt.Println("Error: at least one invoice ID is required")
		return
	}
	if paid && pay != 0 {
		fmt.Println("Error: --pay and --paid cannot be used together")
		return
	}
	if !paid && pay == 0 {
		fmt.Println("Error: provide either --pay or --paid")
		return
	}
	if pay != 0 && len(invoices) != 1 {
		fmt.Println("Error: --pay can only be used with one invoice ID")
		return
	}
	if month < 0 || month > 13 {
		fmt.Println("Error: month must be between 0 and 13")
		return
	}

	_, _, invoiceDB, _, err := get.ConfigData(month)
	if err != nil {
		fmt.Println("Error fetching config data:", err)
		return
	}
	existData := &pkg.JsLocalDB{InvoiceFile: invoiceDB}
	data, err := existData.Invoices()
	if err != nil {
		fmt.Println("Error fetching invoice data:", err)
		return
	}

	var invoiceData []pkg.Invoice
	if err := json.Unmarshal(data, &invoiceData); err != nil {
		fmt.Println("Error parsing invoice data:", err)
		return
	}

	requested := make(map[string]bool, len(invoices))
	for _, invoiceID := range invoices {
		requested[invoiceID] = true
	}
	updated := 0
	for index := range invoiceData {
		if !requested[invoiceData[index].InvoiceID] {
			continue
		}

		if paid {
			invoiceData[index].IsPaid = true
		} else {
			invoiceData[index].PartialPayment += pay
			if invoiceData[index].PartialPayment >= invoiceData[index].Amount {
				invoiceData[index].IsPaid = true
			}
		}
		updated++
	}

	if updated != len(requested) {
		fmt.Println("Error: one or more invoice IDs were not found")
		return
	}

	file, err := os.Create(invoiceDB)
	if err != nil {
		fmt.Println("Error saving invoice data:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(invoiceData); err != nil {
		fmt.Println("Error saving invoice data:", err)
		return
	}
	fmt.Printf("Updated %d invoice(s) successfully\n", updated)
}
